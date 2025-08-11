package storage

import (
	"context"
	"errors"
	"github.com/awakari/subscriptions/model"
	"io"
)

type Storage interface {
	io.Closer
	Create(ctx context.Context, sub model.Subscription) (err error)
	Read(ctx context.Context, interestId, groupId, userId, url string) (cb model.Subscription, err error)
	Update(ctx context.Context, sub model.Subscription, deliveryFailed bool) (err error)
	Delete(ctx context.Context, interestId, groupId, userId, url string) (err error)

	CountByInterest(ctx context.Context, interestId string) (count int64, err error)
	ListByInterest(ctx context.Context, limit uint32, interestId, cursor string) (page []model.Subscription, err error)
	ListByUrl(ctx context.Context, limit uint32, url, cursor string) (page []string, err error)
	ListByUser(ctx context.Context, limit uint32, groupId, userId string) (page []model.Subscription, err error)
	CountAll(ctx context.Context) (count int64, err error)
	ListAll(ctx context.Context, limit uint32, cursor string) (page []model.Subscription, err error)
	ChangeOwner(ctx context.Context, oldGroupId, oldUserId, newGroupId, newUserId string) (n int64, err error)
}

var ErrInternal = errors.New("internal failure")
var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("already exists")
