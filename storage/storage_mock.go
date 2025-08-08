package storage

import (
	"context"
	"github.com/awakari/subscriptions/model"
)

type storageMock struct {
}

func NewStorageMock() Storage {
	return storageMock{}
}

func (s storageMock) Create(ctx context.Context, sub model.Subscription) (err error) {
	//TODO implement me
	panic("implement me")
}

func (s storageMock) Read(ctx context.Context, interestId, groupId, userId, url string) (cb model.Subscription, err error) {
	return
}

func (s storageMock) Update(ctx context.Context, sub model.Subscription) (err error) {
	//TODO implement me
	panic("implement me")
}

func (s storageMock) CountByInterest(ctx context.Context, interestId string) (count int64, err error) {
	count = 42
	return
}

func (s storageMock) Delete(ctx context.Context, interestId, groupId, userId, url string) (err error) {
	//TODO implement me
	panic("implement me")
}

func (s storageMock) ListByInterest(ctx context.Context, limit uint32, interestId, cursor string) (page []model.Subscription, err error) {
	//TODO implement me
	panic("implement me")
}

func (s storageMock) ListByUrl(ctx context.Context, limit uint32, url, cursor string) (page []string, err error) {
	//TODO implement me
	panic("implement me")
}

func (s storageMock) ListByUser(ctx context.Context, limit uint32, groupId, userId string) (page []model.Subscription, err error) {
	//TODO implement me
	panic("implement me")
}

func (s storageMock) CountAll(ctx context.Context) (count int64, err error) {
	//TODO implement me
	panic("implement me")
}

func (s storageMock) ListAll(ctx context.Context, limit uint32, cursor string) (page []model.Subscription, err error) {
	//TODO implement me
	panic("implement me")
}

func (s storageMock) ChangeOwner(ctx context.Context, oldGroupId, oldUserId, newGroupId, newUserId string) (n int64, err error) {
	//TODO implement me
	panic("implement me")
}

func (s storageMock) Close() error {
	//TODO implement me
	panic("implement me")
}
