package service

import (
	"context"
	"fmt"
	"github.com/awakari/subscriptions/model"
	"github.com/awakari/subscriptions/util"
	"log/slog"
)

type logging struct {
	svc Service
	log *slog.Logger
}

func NewLogging(svc Service, log *slog.Logger) Service {
	return logging{
		svc: svc,
		log: log,
	}
}

func (sl logging) Subscribe(ctx context.Context, sub model.Subscription) (err error) {
	err = sl.svc.Subscribe(ctx, sub)
	ll := util.LogLevel(err)
	sl.log.Log(ctx, ll, fmt.Sprintf("service.Subscribe(%+v): err=%s", sub, err))
	return
}

func (sl logging) Subscription(ctx context.Context, interestId, groupId, userId, url string) (sub model.Subscription, err error) {
	sub, err = sl.svc.Subscription(ctx, interestId, groupId, userId, url)
	ll := util.LogLevel(err)
	sl.log.Log(ctx, ll, fmt.Sprintf("service.Subscription(%s, %s, %s, %s): %+v, err=%s", interestId, groupId, userId, url, sub, err))
	return
}

func (sl logging) Unsubscribe(ctx context.Context, interestId, groupId, userId, url string) (err error) {
	err = sl.svc.Unsubscribe(ctx, interestId, groupId, userId, url)
	ll := util.LogLevel(err)
	sl.log.Log(ctx, ll, fmt.Sprintf("service.Unsubscribe(%s, %s, %s, %s): err=%s", interestId, groupId, userId, url, err))
	return
}

func (sl logging) UnsubscribeByInterest(ctx context.Context, interestId string) (count int64, err error) {
	count, err = sl.svc.UnsubscribeByInterest(ctx, interestId)
	ll := util.LogLevel(err)
	sl.log.Log(ctx, ll, fmt.Sprintf("service.UnsubscribeByInterest(%s): %d, err=%s", interestId, count, err))
	return
}

func (sl logging) Update(ctx context.Context, sub model.Subscription) (err error) {
	err = sl.svc.Update(ctx, sub)
	ll := util.LogLevel(err)
	sl.log.Log(ctx, ll, fmt.Sprintf("service.Update(%+v): err=%s", sub, err))
	return
}

func (sl logging) CountByInterest(ctx context.Context, interestId string) (count int64, err error) {
	count, err = sl.svc.CountByInterest(ctx, interestId)
	ll := util.LogLevel(err)
	sl.log.Log(ctx, ll, fmt.Sprintf("service.CountByInterest(%s): %d, err=%s", interestId, count, err))
	return
}

func (sl logging) ListByUrl(ctx context.Context, limit uint32, url, cursor string) (page []string, err error) {
	page, err = sl.svc.ListByUrl(ctx, limit, url, cursor)
	ll := util.LogLevel(err)
	sl.log.Log(ctx, ll, fmt.Sprintf("service.ListByUrl(%d, %s, %s): %d, err=%s", limit, url, cursor, len(page), err))
	return
}

func (sl logging) ListByUser(ctx context.Context, limit uint32, groupId, userId string) (page []model.Subscription, err error) {
	page, err = sl.svc.ListByUser(ctx, limit, groupId, userId)
	ll := util.LogLevel(err)
	sl.log.Log(ctx, ll, fmt.Sprintf("service.ListByUser(%d, %s, %s): %d, err=%s", limit, groupId, userId, len(page), err))
	return
}

func (sl logging) ChangeOwner(ctx context.Context, oldGroupId, oldUserId, newGroupId, newUserId string) (n int64, err error) {
	n, err = sl.svc.ChangeOwner(ctx, oldGroupId, oldUserId, newGroupId, newUserId)
	ll := util.LogLevel(err)
	sl.log.Log(ctx, ll, fmt.Sprintf("service.ChangeOwner(%s/%s => %s/%s): %d, err=%s", oldGroupId, oldUserId, newGroupId, newUserId, n, err))
	return
}

func (sl logging) DeleteExtra(ctx context.Context, limit uint32, groupId, userId string) (n, nDeleted int64, err error) {
	n, nDeleted, err = sl.svc.DeleteExtra(ctx, limit, groupId, userId)
	ll := util.LogLevel(err)
	sl.log.Log(ctx, ll, fmt.Sprintf("service.DeleteExtra(%d, %s, %s): %d, %d, err=%s", limit, groupId, userId, n, nDeleted, err))
	return
}

func (sl logging) RestoreUsagePermits(ctx context.Context) (err error) {
	err = sl.svc.RestoreUsagePermits(ctx)
	ll := util.LogLevel(err)
	sl.log.Log(ctx, ll, fmt.Sprintf("service.RestoreUsageLimits(): err=%s", err))
	return
}
