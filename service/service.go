package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/subscriptions/api/grpc/events"
	"github.com/awakari/subscriptions/api/grpc/usage/permits"
	"github.com/awakari/subscriptions/config"
	"github.com/awakari/subscriptions/model"
	"github.com/awakari/subscriptions/model/usage"
	"github.com/awakari/subscriptions/storage"
	"github.com/cenkalti/backoff/v4"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/segmentio/ksuid"
	"strconv"
	"time"
)

type Service interface {
	Subscribe(ctx context.Context, sub model.Subscription) (err error)
	Subscription(ctx context.Context, interestId, groupId, userId, url string) (sub model.Subscription, err error)
	Unsubscribe(ctx context.Context, interestId, groupId, userId, url string) (err error)
	UnsubscribeByInterest(ctx context.Context, interestId string) (count int64, err error)
	Update(ctx context.Context, sub model.Subscription) (err error)
	ListByInterest(ctx context.Context, limit uint32, interestId, cursor string) (page []model.Subscription, err error)
	ListByUrl(ctx context.Context, limit uint32, url, cursor string) (page []string, err error)
	ListByUser(ctx context.Context, limit uint32, groupId, userId string) (page []model.Subscription, err error)
	DeleteExtra(ctx context.Context, limit uint32, groupId, userId string) (nLeft, nDeleted int64, err error)
	CountByInterest(ctx context.Context, interestId string) (count int64, err error)
	ChangeOwner(ctx context.Context, oldGroupId, oldUserId, newGroupId, newUserId string) (n int64, err error)
	RestoreUsagePermits(ctx context.Context) (err error)
}

type service struct {
	stor       storage.Storage
	svcPermits permits.Service
	svcEvts    events.Service
	cfgEvts    config.EventsConfig
}

const ceKeyCount = "count"

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("already exists")
var ErrInternal = errors.New("internal failure")
var ErrPermitExhausted = errors.New("usage permits exhausted")

func NewService(
	db storage.Storage,
	svcPermits permits.Service,
	svcEvts events.Service,
	cfgEvts config.EventsConfig,
) Service {
	return service{
		stor:       db,
		svcPermits: svcPermits,
		svcEvts:    svcEvts,
		cfgEvts:    cfgEvts,
	}
}

func (svc service) Subscribe(ctx context.Context, sub model.Subscription) (err error) {
	var p usage.Permit
	p, err = svc.svcPermits.Request(ctx, sub.GroupId, sub.UserId, usage.SubjectSubscriptions, 1)
	if err == nil {
		switch p.Count {
		case 0:
			err = ErrPermitExhausted
		default:
			err = svc.stor.Create(ctx, sub)
			switch {
			case errors.Is(err, storage.ErrConflict):
				err = fmt.Errorf("%w: %s", ErrConflict, err)
				err = errors.Join(err, svc.svcPermits.Release(ctx, sub.GroupId, p.UserId, usage.SubjectSubscriptions, 1))
			case err != nil:
				err = fmt.Errorf("%w: %s", ErrInternal, err)
				err = errors.Join(err, svc.svcPermits.Release(ctx, sub.GroupId, p.UserId, usage.SubjectSubscriptions, 1))
			default:
				go svc.notifyFollowersCount(ctx, sub.InterestId)
			}
		}
	}
	return
}

func (svc service) Subscription(ctx context.Context, interestId, groupId, userId, url string) (cb model.Subscription, err error) {
	cb, err = svc.stor.Read(ctx, interestId, groupId, userId, url)
	switch {
	case errors.Is(err, storage.ErrNotFound):
		err = fmt.Errorf("%w: %s, %s", ErrNotFound, interestId, url)
	case err != nil:
		err = fmt.Errorf("%w: %s, %s, %s", ErrInternal, interestId, url, err)
	}
	return
}

func (svc service) Unsubscribe(ctx context.Context, interestId, groupId, userId, url string) (err error) {
	err = svc.stor.Delete(ctx, interestId, groupId, userId, url)
	switch {
	case errors.Is(err, storage.ErrNotFound):
		err = fmt.Errorf("%w: %s", ErrNotFound, err)
	case err != nil:
		err = fmt.Errorf("%w: %s", ErrInternal, err)
	default:
		err = svc.svcPermits.Release(ctx, groupId, userId, usage.SubjectSubscriptions, 1)
		go svc.notifyFollowersCount(ctx, interestId)
	}
	return
}

func (svc service) UnsubscribeByInterest(ctx context.Context, interestId string) (count int64, err error) {
	var cursor string
	var page []model.Subscription
	for {
		page, err = svc.stor.ListByInterest(ctx, 10, interestId, cursor)
		if err != nil || len(page) == 0 {
			break
		}
		cursor = page[len(page)-1].Url
		for _, sub := range page {
			err = svc.stor.Delete(ctx, interestId, sub.GroupId, sub.UserId, sub.Url)
			switch {
			case err == nil:
				err = svc.svcPermits.Release(ctx, sub.GroupId, sub.UserId, usage.SubjectSubscriptions, 1)
			case errors.Is(err, storage.ErrNotFound):
				err = nil
			}
			if err != nil {
				break
			}
		}
		if err != nil {
			break
		}
	}
	go svc.notifyFollowersCount(ctx, interestId)
	return
}

func (svc service) Update(ctx context.Context, sub model.Subscription) (err error) {
	err = svc.stor.Update(ctx, sub)
	switch {
	case errors.Is(err, storage.ErrNotFound):
		err = fmt.Errorf("%w: %s, %s", ErrNotFound, sub.InterestId, sub.Url)
	case err != nil:
		err = fmt.Errorf("%w: %s, %s, %s", ErrInternal, sub.InterestId, sub.Url, err)
	}
	return
}

func (svc service) ListByInterest(ctx context.Context, limit uint32, interestId, cursor string) (page []model.Subscription, err error) {
	page, err = svc.stor.ListByInterest(ctx, limit, interestId, cursor)
	if err != nil {
		err = fmt.Errorf("%w, interest: %s, %s", ErrInternal, interestId, err)
	}
	return
}

func (svc service) CountByInterest(ctx context.Context, interestId string) (count int64, err error) {
	count, err = svc.stor.CountByInterest(ctx, interestId)
	if err != nil {
		err = fmt.Errorf("%w, interest: %s, %s", ErrInternal, interestId, err)
	}
	return
}

func (svc service) ListByUrl(ctx context.Context, limit uint32, url, cursor string) (page []string, err error) {
	page, err = svc.stor.ListByUrl(ctx, limit, url, cursor)
	if err != nil {
		err = fmt.Errorf("%w, url: %s, cursor: %s, %s", ErrInternal, url, cursor, err)
	}
	return
}

func (svc service) ListByUser(ctx context.Context, limit uint32, groupId, userId string) (page []model.Subscription, err error) {
	page, err = svc.stor.ListByUser(ctx, limit, groupId, userId)
	return
}

func (svc service) ChangeOwner(ctx context.Context, oldGroupId, oldUserId, newGroupId, newUserId string) (n int64, err error) {
	n, err = svc.stor.ChangeOwner(ctx, oldGroupId, oldUserId, newGroupId, newUserId)
	return
}

func (svc service) DeleteExtra(ctx context.Context, limit uint32, groupId, userId string) (nLeft, nDeleted int64, err error) {

	// list until it gets all results in one page
	limitList := limit
	var page []model.Subscription
	for {
		page, err = svc.stor.ListByUser(ctx, limitList, groupId, userId)
		if err != nil {
			break
		}
		if len(page) < int(limitList) {
			break
		}
		limitList *= 2 // try to increase the list limit
	}

	if err == nil {
		nLeft = int64(len(page))
		if len(page) > int(limit) {
			for _, sub := range page[limit:] {
				err = svc.Unsubscribe(ctx, sub.InterestId, groupId, userId, sub.Url)
				if err != nil {
					break
				}
				nDeleted++
			}
			nLeft -= nDeleted
		}
	}

	return
}

func (svc service) notifyFollowersCount(ctx context.Context, interestId string) (err error) {
	var count int64
	count, err = svc.CountByInterest(ctx, interestId)
	if err == nil {
		err = svc.notifyFollowersCountOnce(ctx, interestId, count)
		if err != nil {
			b := backoff.NewExponentialBackOff()
			err = backoff.Retry(func() error {
				return svc.notifyFollowersCountOnce(ctx, interestId, count)
			}, b)
		}
	}
	return
}

func (svc service) notifyFollowersCountOnce(ctx context.Context, interestId string, count int64) (err error) {
	p, _ := svc.svcEvts.NewPublisher(ctx, svc.cfgEvts.FollowersChanged.Topic)
	if p != nil {
		defer p.Close()
		var ackCount uint32
		ackCount, err = p.Write(ctx, []*pb.CloudEvent{
			{
				Id:          ksuid.New().String(),
				Source:      svc.cfgEvts.FollowersChanged.Source,
				SpecVersion: model.CeSpecVersion,
				Type:        svc.cfgEvts.FollowersChanged.Topic,
				Attributes: map[string]*pb.CloudEventAttributeValue{
					ceKeyCount: {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: strconv.FormatInt(count, 10),
						},
					},
				},
				Data: &pb.CloudEvent_TextData{
					TextData: interestId,
				},
			},
		})
		if err == nil && ackCount < 1 {
			err = fmt.Errorf("notification publication not acknowledged: interestId=%s", interestId)
		}
	}
	return
}

func (svc service) RestoreUsagePermits(ctx context.Context) (err error) {
	var cursor string
	for {
		var page []model.Subscription
		page, err = svc.stor.ListAll(ctx, 100, cursor)
		fmt.Printf("RestoreUsagePermits: next subs page: %d, %s\n", len(page), err)
		if err != nil {
			break
		}
		if len(page) == 0 {
			break
		}
		cursor = page[len(page)-1].InternalId
		for _, sub := range page {
			if sub.UserId != "" {
				var pageUserAll []model.Subscription
				pageUserAll, err = svc.stor.ListByUser(ctx, 100, sub.GroupId, sub.UserId)
				fmt.Printf("RestoreUsagePermits: user: %s, subscriptions: %d, %s\n", sub.UserId, len(pageUserAll), err)
				if err != nil {
					break
				}
				var u usage.Usage
				err = svc.svcPermits.GetUsage(ctx, sub.GroupId, sub.UserId, usage.SubjectSubscriptions, &u)
				if err != nil {
					break
				}
				diff := int(u.Count) - len(pageUserAll)
				fmt.Printf("RestoreUsagePermits: user: %s, diff: %d\n", sub.UserId, diff)
				switch {
				case diff < 0:
					_, err = svc.svcPermits.Request(ctx, sub.GroupId, sub.UserId, usage.SubjectSubscriptions, uint32(-diff))
				case diff > 0:
					err = svc.svcPermits.Release(ctx, sub.GroupId, sub.UserId, usage.SubjectSubscriptions, uint32(diff))
				}
				if err != nil {
					break
				}
				time.Sleep(1 * time.Second)
			}
		}
		if err != nil {
			break
		}
	}
	return
}
