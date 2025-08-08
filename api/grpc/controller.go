package grpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/subscriptions/model"
	"github.com/awakari/subscriptions/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io"
)

type controller struct {
	svc service.Service
}

func NewController(svc service.Service) ServiceServer {
	return controller{
		svc: svc,
	}
}

func (c controller) Update(ctx context.Context, req *UpdateRequest) (resp *UpdateResponse, err error) {
	resp = &UpdateResponse{}
	err = c.svc.Update(ctx, model.Subscription{
		InterestId:   req.InterestId,
		Url:          req.Url,
		GroupId:      req.GroupId,
		UserId:       req.UserId,
		LastResultAt: req.LastResultAt.AsTime(),
	})
	err = encodeError(err)
	return
}

func (c controller) Delete(ctx context.Context, req *DeleteRequest) (resp *DeleteResponse, err error) {
	resp = &DeleteResponse{}
	err = c.svc.Unsubscribe(ctx, req.InterestId, req.GroupId, req.UserId, req.Url)
	err = encodeError(err)
	return
}

func (c controller) ListByInterest(ctx context.Context, req *ListByInterestRequest) (resp *ListByInterestResponse, err error) {
	resp = &ListByInterestResponse{}
	var page []model.Subscription
	page, err = c.svc.ListByInterest(ctx, req.Limit, req.InterestId, req.Cursor)
	if err == nil {
		for _, sub := range page {
			var f Format
			switch sub.Format {
			case model.FormatRss:
				f = Format_Rss
			case model.FormatCeJs:
				f = Format_CeJs
			case model.FormatHtml:
				f = Format_Html
			default:
				err = errors.New(fmt.Sprintf("unknown subscription format: %s", sub.Format))
			}
			resp.Page = append(resp.Page, &Subscription{
				GroupId:      sub.GroupId,
				UserId:       sub.UserId,
				Url:          sub.Url,
				Format:       f,
				Secret:       sub.Secret,
				LastResultAt: timestamppb.New(sub.LastResultAt),
				IntervalMin:  durationpb.New(sub.IntervalMin),
			})
		}
	}
	err = encodeError(err)
	return
}

func (c controller) Count(ctx context.Context, req *CountRequest) (resp *CountResponse, err error) {
	resp = &CountResponse{}
	switch {
	case req.Filter.Id != "":
		resp.Count, err = c.svc.CountByInterest(ctx, req.Filter.Id)
	case req.Filter.GroupId != "" && req.Filter.UserId != "":
		var page []model.Subscription
		page, err = c.svc.ListByUser(ctx, req.Limit, req.Filter.GroupId, req.Filter.UserId)
		if err == nil {
			resp.Count = int64(len(page))
		}
	}
	err = encodeError(err)
	return
}

func (c controller) DeleteExtra(ctx context.Context, req *DeleteExtraRequest) (resp *DeleteExtraResponse, err error) {
	resp = &DeleteExtraResponse{}
	if req.Filter.GroupId != "" && req.Filter.UserId != "" {
		resp.CountLeft, resp.CountDeleted, err = c.svc.DeleteExtra(ctx, req.Limit, req.Filter.GroupId, req.Filter.UserId)
	}
	err = encodeError(err)
	return
}

func (c controller) ChangeOwner(ctx context.Context, req *ChangeOwnerRequest) (resp *ChangeOwnerResponse, err error) {
	resp = &ChangeOwnerResponse{}
	resp.N, err = c.svc.ChangeOwner(ctx, req.OldGroupId, req.OldUserId, req.NewGroupId, req.NewUserId)
	err = encodeError(err)
	return
}

func encodeError(src error) (dst error) {
	switch {
	case src == nil:
	case src == io.EOF:
		dst = src
	case errors.Is(src, context.DeadlineExceeded):
		dst = src
	case errors.Is(src, service.ErrNotFound):
		dst = status.Error(codes.NotFound, src.Error())
	case errors.Is(src, service.ErrInternal):
		dst = status.Error(codes.Internal, src.Error())
	default:
		dst = status.Error(codes.Unknown, src.Error())
	}
	return
}
