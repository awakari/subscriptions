package storage

import (
	"context"
	"fmt"
	"github.com/awakari/subscriptions/model"
	"github.com/bytedance/sonic"
	"github.com/go-redis/cache/v9"
	"log/slog"
	"time"
)

type storageCache struct {
	stor     Storage
	cache    *cache.Cache
	cacheTtl time.Duration
	log      *slog.Logger
}

type cacheValue struct {
	GroupId      string        `json:"group_id,omitempty"`
	UserId       string        `json:"user_id,omitempty"`
	Url          string        `json:"url"`
	Secret       []byte        `json:"secret"`
	Format       model.Format  `json:"format"`
	IntervalMin  time.Duration `json:"interval,omitempty"`
	LastResultAt time.Time     `json:"last,omitempty"`
	ErrorCount   uint32        `json:"error_count,omitempty"`
}

type cacheValuePage struct {
	Page []cacheValue `bson:"page"`
}

type cacheValueBytes struct {
	Bytes []byte
}

const keyPrefix = "subscription:"
const keyPrefixPages = "subscriptions:"

func NewCache(stor Storage, cache *cache.Cache, cacheTtl time.Duration, log *slog.Logger) Storage {
	return storageCache{
		stor:     stor,
		cache:    cache,
		cacheTtl: cacheTtl,
		log:      log,
	}
}

func (c storageCache) Close() error {
	return c.stor.Close()
}

func (c storageCache) Create(ctx context.Context, sub model.Subscription) (err error) {
	err = c.stor.Create(ctx, sub)
	if err == nil {
		err = c.invalidatePages(ctx, sub.InterestId)
		if err != nil {
			c.log.Debug(fmt.Sprintf("create: pages cache failure: invalidate by %s: %s", sub.InterestId, err))
		}
		cv := new(cacheValueBytes)
		cv.Bytes, err = sonic.Marshal(&cacheValue{
			GroupId:     sub.GroupId,
			UserId:      sub.UserId,
			Url:         sub.Url,
			Secret:      sub.Secret,
			Format:      sub.Format,
			IntervalMin: sub.IntervalMin,
		})
		k := key(sub.InterestId, sub.Url)
		if err == nil {
			item := &cache.Item{
				Ctx:   ctx,
				Key:   k,
				Value: cv,
				TTL:   c.cacheTtl,
			}
			err = c.cache.Set(item)
		}
		if err != nil {
			c.log.Debug(fmt.Sprintf("cache failure: put key %s: %s", k, err))
			err = nil
		}
	}
	return
}

func (c storageCache) Update(ctx context.Context, sub model.Subscription, deliveryFailed bool) (err error) {
	err = c.stor.Update(ctx, sub, deliveryFailed)
	if err == nil {
		err = c.invalidatePages(ctx, sub.InterestId)
		if err != nil {
			c.log.Debug(fmt.Sprintf("update: pages cache failure: invalidate by %s: %s", sub.InterestId, err))
		}
		k := key(sub.InterestId, sub.Url)
		err = c.cache.Delete(ctx, k)
		if err != nil {
			c.log.Debug(fmt.Sprintf("cache failure: delete key %s: %s", k, err))
			err = nil
		}
	}
	return
}

func (c storageCache) Read(ctx context.Context, interestId, groupId, userId, url string) (cb model.Subscription, err error) {
	k := key(interestId, url)
	val := new(cacheValueBytes)
	load := func(_ *cache.Item) (result any, err error) {
		subLoad, errLoad := c.stor.Read(ctx, interestId, groupId, userId, url)
		if errLoad == nil {
			cv := &cacheValueBytes{}
			cv.Bytes, errLoad = sonic.Marshal(&cacheValue{
				GroupId:      subLoad.GroupId,
				UserId:       subLoad.UserId,
				Url:          subLoad.Url,
				Secret:       subLoad.Secret,
				Format:       subLoad.Format,
				IntervalMin:  subLoad.IntervalMin,
				LastResultAt: subLoad.LastResultAt,
				ErrorCount:   subLoad.ErrorCount,
			})
			result = cv
		}
		return
	}
	item := &cache.Item{
		Ctx:   ctx,
		Key:   k,
		Value: val,
		TTL:   c.cacheTtl,
		Do:    load,
		SetNX: true,
	}
	err = c.cache.Once(item)
	var cv cacheValue
	if err == nil {
		err = sonic.Unmarshal(item.Value.(*cacheValueBytes).Bytes, &cv)
	}
	switch err {
	case nil:
		cb.Url = cv.Url
		cb.Secret = cv.Secret
		cb.Format = cv.Format
	default:
		c.log.Debug(fmt.Sprintf("cache failure: get key %s: %s", k, err))
		cb, err = c.stor.Read(ctx, interestId, groupId, userId, url) // fallback
	}
	return
}

func (c storageCache) CountByInterest(ctx context.Context, interestId string) (count int64, err error) {
	count, err = c.stor.CountByInterest(ctx, interestId)
	return
}

func (c storageCache) Delete(ctx context.Context, interestId, groupId, userId, url string) (err error) {
	err = c.stor.Delete(ctx, interestId, groupId, userId, url)
	if err == nil {
		err = c.invalidatePages(ctx, interestId)
		if err != nil {
			c.log.Warn(fmt.Sprintf("delete: pages cache failure: invalidate by %s: %s", interestId, err))
		}
		err = c.cache.Delete(ctx, key(interestId, url))
		if err != nil {
			c.log.Debug(fmt.Sprintf("cache failure: del key %s: %s", key(interestId, url), err))
			err = nil
		}
	}
	return
}

func (c storageCache) ListByInterest(ctx context.Context, limit uint32, interestId, cursor string) (page []model.Subscription, err error) {
	k := keyPages(interestId, cursor)
	v := new(cacheValueBytes)
	load := func(_ *cache.Item) (result any, err error) {
		vLoad, errLoad := c.stor.ListByInterest(ctx, limit, interestId, cursor)
		if errLoad == nil {
			cvp := cacheValuePage{}
			for _, vItem := range vLoad {
				cvp.Page = append(cvp.Page, cacheValue{
					GroupId:      vItem.GroupId,
					UserId:       vItem.UserId,
					Url:          vItem.Url,
					Secret:       vItem.Secret,
					Format:       vItem.Format,
					IntervalMin:  vItem.IntervalMin,
					LastResultAt: vItem.LastResultAt,
					ErrorCount:   vItem.ErrorCount,
				})
			}
			cv := &cacheValueBytes{}
			cv.Bytes, errLoad = sonic.Marshal(&cvp)
			result = cv
		}
		fmt.Printf("Cache miss: key=%s, loaded bytes=%s, err=%s\n", k, result, errLoad)
		return
	}
	item := &cache.Item{
		Ctx:   ctx,
		Key:   k,
		Value: v,
		TTL:   c.cacheTtl,
		Do:    load,
		SetNX: true,
	}
	err = c.cache.Once(item)
	var cvp cacheValuePage
	if err == nil {
		err = sonic.Unmarshal(item.Value.(*cacheValueBytes).Bytes, &cvp)
	}
	switch {
	case err == nil:
		for _, cv := range cvp.Page {
			page = append(page, model.Subscription{
				GroupId:      cv.GroupId,
				UserId:       cv.UserId,
				Url:          cv.Url,
				Secret:       cv.Secret,
				Format:       cv.Format,
				IntervalMin:  cv.IntervalMin,
				LastResultAt: cv.LastResultAt,
			})
		}
	default:
		c.log.Debug(fmt.Sprintf("cache failure: get key %s: %s", k, err))
		page, err = c.stor.ListByInterest(ctx, limit, interestId, cursor) // fallback
	}
	return
}

func (c storageCache) ListByUrl(ctx context.Context, limit uint32, url, cursor string) (page []string, err error) {
	page, err = c.stor.ListByUrl(ctx, limit, url, cursor)
	return
}

func (c storageCache) ListByUser(ctx context.Context, limit uint32, groupId, userId string) (page []model.Subscription, err error) {
	page, err = c.stor.ListByUser(ctx, limit, groupId, userId)
	return
}

func (c storageCache) CountAll(ctx context.Context) (count int64, err error) {
	count, err = c.stor.CountAll(ctx)
	return
}

func (c storageCache) ListAll(ctx context.Context, limit uint32, cursor string) (page []model.Subscription, err error) {
	page, err = c.stor.ListAll(ctx, limit, cursor)
	return
}

func (c storageCache) ChangeOwner(ctx context.Context, oldGroupId, oldUserId, newGroupId, newUserId string) (n int64, err error) {
	n, err = c.stor.ChangeOwner(ctx, oldGroupId, oldUserId, newGroupId, newUserId)
	return
}

func (c storageCache) invalidatePages(ctx context.Context, interestId string) (err error) {
	var cursor string
	for {
		k := keyPages(interestId, cursor)
		v := new(cacheValueBytes)
		err = c.cache.Get(ctx, k, v)
		var cvp cacheValuePage
		if err == nil {
			err = sonic.Unmarshal(v.Bytes, &cvp)
		}
		if err == nil {
			if len(cvp.Page) > 0 {
				err = c.cache.Delete(ctx, k)
				cursor = cvp.Page[len(cvp.Page)-1].Url
			} else {
				break
			}
		}
		if err != nil {
			err = fmt.Errorf("%w: %s", err, k)
			break
		}
	}
	return
}

func key(interestId, url string) (k string) {
	k = keyPrefix + interestId + ":" + url
	return
}

func keyPages(interestId, cursor string) (k string) {
	k = keyPrefixPages + interestId + ":" + cursor
	return
}
