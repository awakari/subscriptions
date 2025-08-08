package storage

import (
	"context"
	"fmt"
	"github.com/awakari/subscriptions/config"
	"github.com/awakari/subscriptions/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

var (
	dbUri = os.Getenv("DB_URI_TEST_MONGO")
)

func TestNewStorage(t *testing.T) {
	//
	collName := fmt.Sprintf("callbacks-test-%d", time.Now().UnixMicro())
	dbCfg := config.DbConfig{
		Uri:  dbUri,
		Name: "subscriptions",
	}
	dbCfg.Table.Name = collName
	dbCfg.Table.Shard = false
	dbCfg.Tls.Enabled = true
	dbCfg.Tls.Insecure = true
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	s, err := NewStorage(ctx, dbCfg)
	assert.NotNil(t, s)
	assert.Nil(t, err)
	//
	clear(ctx, t, s.(storageMongo))
}

func clear(ctx context.Context, t *testing.T, s storageMongo) {
	require.Nil(t, s.coll.Drop(ctx))
	require.Nil(t, s.Close())
}

func TestStorageMongo_Create(t *testing.T) {
	//
	collName := fmt.Sprintf("callbacks-test-%d", time.Now().UnixMicro())
	dbCfg := config.DbConfig{
		Uri:  dbUri,
		Name: "subscriptions",
	}
	dbCfg.Table.Name = collName
	dbCfg.Tls.Enabled = true
	dbCfg.Tls.Insecure = true
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()
	s, err := NewStorage(ctx, dbCfg)
	require.Nil(t, err)
	assert.NotNil(t, s)
	//
	defer clear(ctx, t, s.(storageMongo))
	//
	err = s.Create(ctx, model.Subscription{
		InterestId: "sub0",
		GroupId:    "group0",
		UserId:     "user0",
		Url:        "url0",
	})
	require.Nil(t, err)
	//
	cases := map[string]struct {
		src model.Subscription
		err error
	}{
		"ok": {
			src: model.Subscription{
				InterestId:  "sub0",
				GroupId:     "group0",
				UserId:      "user0",
				Url:         "url1",
				Secret:      []byte("secret1"),
				Format:      model.FormatCeJs,
				IntervalMin: 1 * time.Second,
			},
		},
		"conflict": {
			src: model.Subscription{
				InterestId:  "sub0",
				Url:         "url0",
				Secret:      []byte("secret1"),
				Format:      model.FormatCeJs,
				IntervalMin: 1 * time.Second,
			},
			err: ErrConflict,
		},
	}
	//
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			err = s.Create(ctx, c.src)
			assert.ErrorIs(t, err, c.err)
		})
	}
}

func TestStorageMongo_Read(t *testing.T) {
	//
	collName := fmt.Sprintf("callbacks-test-%d", time.Now().UnixMicro())
	dbCfg := config.DbConfig{
		Uri:  dbUri,
		Name: "subscriptions",
	}
	dbCfg.Table.Name = collName
	dbCfg.Tls.Enabled = true
	dbCfg.Tls.Insecure = true
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()
	s, err := NewStorage(ctx, dbCfg)
	require.Nil(t, err)
	assert.NotNil(t, s)
	//
	defer clear(ctx, t, s.(storageMongo))
	//
	err = s.Create(ctx, model.Subscription{
		InterestId:  "sub0",
		Url:         "url0",
		GroupId:     "group0",
		UserId:      "user0",
		Secret:      []byte{1, 2, 3},
		Format:      model.FormatCeJs,
		IntervalMin: 1 * time.Minute,
	})
	require.Nil(t, err)
	err = s.Update(ctx, model.Subscription{
		InterestId:   "sub0",
		Url:          "url0",
		GroupId:      "group0",
		UserId:       "user0",
		Secret:       []byte{1, 2, 3},
		Format:       model.FormatCeJs,
		IntervalMin:  1 * time.Minute,
		LastResultAt: time.Date(2025, 2, 28, 10, 59, 35, 0, time.UTC),
	})
	require.Nil(t, err)
	//
	cases := map[string]struct {
		subId   string
		groupId string
		userId  string
		url     string
		out     model.Subscription
		err     error
	}{
		"ok": {
			subId:   "sub0",
			url:     "url0",
			groupId: "group0",
			userId:  "user0",
			out: model.Subscription{
				GroupId:      "group0",
				UserId:       "user0",
				Url:          "url0",
				Secret:       []byte{1, 2, 3},
				Format:       model.FormatCeJs,
				IntervalMin:  1 * time.Minute,
				LastResultAt: time.Date(2025, 2, 28, 10, 59, 35, 0, time.UTC),
			},
		},
		"not found": {
			subId: "sub0",
			url:   "url1",
			err:   ErrNotFound,
		},
	}
	//
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			var out model.Subscription
			out, err = s.Read(ctx, c.subId, c.groupId, c.userId, c.url)
			assert.Equal(t, c.out, out)
			assert.ErrorIs(t, err, c.err)
		})
	}
}

func TestStorageMongo_Update(t *testing.T) {
	//
	collName := fmt.Sprintf("callbacks-test-%d", time.Now().UnixMicro())
	dbCfg := config.DbConfig{
		Uri:  dbUri,
		Name: "subscriptions",
	}
	dbCfg.Table.Name = collName
	dbCfg.Tls.Enabled = true
	dbCfg.Tls.Insecure = true
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()
	s, err := NewStorage(ctx, dbCfg)
	require.Nil(t, err)
	assert.NotNil(t, s)
	//
	defer clear(ctx, t, s.(storageMongo))
	//
	err = s.Create(ctx, model.Subscription{
		InterestId:  "sub0",
		GroupId:     "group0",
		UserId:      "user0",
		Url:         "url0",
		Format:      model.FormatRss,
		IntervalMin: 1 * time.Second,
	})
	require.Nil(t, err)
	//
	cases := map[string]struct {
		src model.Subscription
		err error
	}{
		"missing": {
			src: model.Subscription{
				InterestId:   "sub0",
				GroupId:      "group0",
				UserId:       "user0",
				Url:          "url1",
				Secret:       []byte("secret1"),
				Format:       model.FormatCeJs,
				IntervalMin:  1 * time.Minute,
				LastResultAt: time.Date(2025, 2, 28, 10, 59, 35, 0, time.UTC),
			},
			err: ErrNotFound,
		},
		"ok": {
			src: model.Subscription{
				InterestId:   "sub0",
				GroupId:      "group0",
				UserId:       "user0",
				Url:          "url0",
				Secret:       []byte("secret1"),
				Format:       model.FormatCeJs,
				IntervalMin:  1 * time.Minute,
				LastResultAt: time.Date(2025, 2, 28, 10, 59, 35, 0, time.UTC),
			},
		},
		"invalid user id": {
			src: model.Subscription{
				InterestId:   "sub0",
				GroupId:      "group0",
				UserId:       "user1",
				Url:          "url0",
				Secret:       []byte("secret1"),
				Format:       model.FormatCeJs,
				IntervalMin:  1 * time.Minute,
				LastResultAt: time.Date(2025, 2, 28, 10, 59, 35, 0, time.UTC),
			},
			err: ErrNotFound,
		},
	}
	//
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			err = s.Update(ctx, c.src)
			assert.ErrorIs(t, err, c.err)
		})
	}
}

func TestStorageMongo_CountByInterest(t *testing.T) {
	//
	collName := fmt.Sprintf("callbacks-test-%d", time.Now().UnixMicro())
	dbCfg := config.DbConfig{
		Uri:  dbUri,
		Name: "subscriptions",
	}
	dbCfg.Table.Name = collName
	dbCfg.Tls.Enabled = true
	dbCfg.Tls.Insecure = true
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()
	s, err := NewStorage(ctx, dbCfg)
	require.Nil(t, err)
	assert.NotNil(t, s)
	//
	defer clear(ctx, t, s.(storageMongo))
	//
	err = s.Create(ctx, model.Subscription{
		InterestId: "sub1",
		Url:        "url0",
		Secret:     []byte{1, 2, 3},
		Format:     model.FormatCeJs,
	})
	require.Nil(t, err)
	err = s.Create(ctx, model.Subscription{
		InterestId: "sub2",
		Url:        "url1",
		Secret:     []byte{1, 2, 3},
		Format:     model.FormatCeJs,
	})
	require.Nil(t, err)
	err = s.Create(ctx, model.Subscription{
		InterestId: "sub2",
		Url:        "url2",
		Secret:     []byte{1, 2, 3},
		Format:     model.FormatCeJs,
	})
	require.Nil(t, err)
	//
	cases := map[string]struct {
		subId string
		count int64
		err   error
	}{
		"0": {},
		"1": {
			subId: "sub1",
			count: 1,
		},
		"2": {
			subId: "sub2",
			count: 2,
		},
	}
	//
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			count, err := s.CountByInterest(ctx, c.subId)
			assert.Equal(t, c.count, count)
			assert.ErrorIs(t, err, c.err)
		})
	}
}

func TestStorageMongo_Delete(t *testing.T) {
	//
	collName := fmt.Sprintf("callbacks-test-%d", time.Now().UnixMicro())
	dbCfg := config.DbConfig{
		Uri:  dbUri,
		Name: "subscriptions",
	}
	dbCfg.Table.Name = collName
	dbCfg.Tls.Enabled = true
	dbCfg.Tls.Insecure = true
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()
	s, err := NewStorage(ctx, dbCfg)
	require.Nil(t, err)
	assert.NotNil(t, s)
	//
	defer clear(ctx, t, s.(storageMongo))
	//
	err = s.Create(ctx, model.Subscription{
		InterestId: "sub0",
		Url:        "url0",
		Secret:     []byte{1, 2, 3},
	})
	require.Nil(t, err)
	//
	cases := map[string]struct {
		subId   string
		groupId string
		userId  string
		url     string
		err     error
	}{
		"ok": {
			subId: "sub0",
			url:   "url0",
		},
		"not found": {
			subId: "sub1",
			url:   "url0",
			err:   ErrNotFound,
		},
	}
	//
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			err = s.Delete(ctx, c.subId, c.groupId, c.userId, c.url)
			assert.ErrorIs(t, err, c.err)
		})
	}
}

func TestStorageMongo_ListByInterest(t *testing.T) {
	//
	collName := fmt.Sprintf("callbacks-test-%d", time.Now().UnixMicro())
	dbCfg := config.DbConfig{
		Uri:  dbUri,
		Name: "subscriptions",
	}
	dbCfg.Table.Name = collName
	dbCfg.Tls.Enabled = true
	dbCfg.Tls.Insecure = true
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()
	s, err := NewStorage(ctx, dbCfg)
	require.Nil(t, err)
	assert.NotNil(t, s)
	//
	defer clear(ctx, t, s.(storageMongo))
	//
	err = s.Create(ctx, model.Subscription{
		InterestId:  "sub0",
		Url:         "url0",
		GroupId:     "group0",
		UserId:      "user0",
		Secret:      []byte{1, 2, 3},
		Format:      model.FormatCeJs,
		IntervalMin: 1 * time.Minute,
	})
	require.Nil(t, err)
	err = s.Update(ctx, model.Subscription{
		InterestId:   "sub0",
		Url:          "url0",
		GroupId:      "group0",
		UserId:       "user0",
		Secret:       []byte{1, 2, 3},
		Format:       model.FormatCeJs,
		IntervalMin:  1 * time.Minute,
		LastResultAt: time.Date(2025, 2, 28, 10, 59, 35, 0, time.UTC),
	})
	require.Nil(t, err)
	err = s.Create(ctx, model.Subscription{
		InterestId:  "sub0",
		Url:         "url1",
		Secret:      []byte{1, 2, 3},
		Format:      model.FormatCeJs,
		IntervalMin: 1 * time.Hour,
	})
	require.Nil(t, err)
	err = s.Create(ctx, model.Subscription{
		InterestId: "sub0",
		Url:        "url2",
		Secret:     []byte{1, 2, 3},
		Format:     model.FormatCeJs,
	})
	require.Nil(t, err)
	//
	cases := map[string]struct {
		subId  string
		cursor string
		limit  uint32
		out    []model.Subscription
		err    error
	}{
		"w/ limit": {
			subId: "sub0",
			limit: 2,
			out: []model.Subscription{
				{
					GroupId:      "group0",
					UserId:       "user0",
					Url:          "url0",
					Secret:       []byte{1, 2, 3},
					Format:       model.FormatCeJs,
					IntervalMin:  1 * time.Minute,
					LastResultAt: time.Date(2025, 2, 28, 10, 59, 35, 0, time.UTC),
				},
				{
					Url:         "url1",
					Secret:      []byte{1, 2, 3},
					Format:      model.FormatCeJs,
					IntervalMin: 1 * time.Hour,
				},
			},
		},
		"w/ cursor": {
			subId:  "sub0",
			cursor: "url1",
			out: []model.Subscription{
				{
					Url:    "url2",
					Secret: []byte{1, 2, 3},
					Format: model.FormatCeJs,
				},
			},
		},
	}
	//
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			var out []model.Subscription
			out, err = s.ListByInterest(ctx, c.limit, c.subId, c.cursor)
			assert.Equal(t, c.out, out)
			assert.ErrorIs(t, err, c.err)
		})
	}
}

func TestStorageMongo_ListByUrl(t *testing.T) {
	//
	collName := fmt.Sprintf("callbacks-test-%d", time.Now().UnixMicro())
	dbCfg := config.DbConfig{
		Uri:  dbUri,
		Name: "subscriptions",
	}
	dbCfg.Table.Name = collName
	dbCfg.Tls.Enabled = true
	dbCfg.Tls.Insecure = true
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()
	s, err := NewStorage(ctx, dbCfg)
	require.Nil(t, err)
	assert.NotNil(t, s)
	//
	defer clear(ctx, t, s.(storageMongo))
	//
	err = s.Create(ctx, model.Subscription{
		InterestId: "sub0",
		Url:        "url0",
		Secret:     []byte{1, 2, 3},
		Format:     model.FormatCeJs,
	})
	require.Nil(t, err)
	err = s.Create(ctx, model.Subscription{
		InterestId: "sub1",
		Url:        "url0",
		Secret:     []byte{1, 2, 3},
		Format:     model.FormatCeJs,
	})
	require.Nil(t, err)
	err = s.Create(ctx, model.Subscription{
		InterestId: "sub2",
		Url:        "url0",
		Secret:     []byte{1, 2, 3},
		Format:     model.FormatCeJs,
	})
	require.Nil(t, err)
	//
	cases := map[string]struct {
		url    string
		cursor string
		limit  uint32
		out    []string
		err    error
	}{
		"w/ limit": {
			url:   "url0",
			limit: 2,
			out: []string{
				"sub0",
				"sub1",
			},
		},
		"w/ cursor": {
			url:    "url0",
			cursor: "sub1",
			out: []string{
				"sub2",
			},
		},
	}
	//
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			var out []string
			out, err = s.ListByUrl(ctx, c.limit, c.url, c.cursor)
			assert.Equal(t, c.out, out)
			assert.ErrorIs(t, err, c.err)
		})
	}
}

func TestStorageMongo_ListByUser(t *testing.T) {
	//
	collName := fmt.Sprintf("callbacks-test-%d", time.Now().UnixMicro())
	dbCfg := config.DbConfig{
		Uri:  dbUri,
		Name: "subscriptions",
	}
	dbCfg.Table.Name = collName
	dbCfg.Tls.Enabled = true
	dbCfg.Tls.Insecure = true
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()
	s, err := NewStorage(ctx, dbCfg)
	require.Nil(t, err)
	assert.NotNil(t, s)
	//
	defer clear(ctx, t, s.(storageMongo))
	//
	err = s.Create(ctx, model.Subscription{
		InterestId: "sub0",
		GroupId:    "group0",
		UserId:     "user0",
		Url:        "url0",
		Secret:     []byte{1, 2, 3},
		Format:     model.FormatCeJs,
	})
	require.Nil(t, err)
	err = s.Create(ctx, model.Subscription{
		InterestId: "sub1",
		GroupId:    "group0",
		UserId:     "user0",
		Url:        "url0",
		Secret:     []byte{1, 2, 3},
		Format:     model.FormatCeJs,
	})
	require.Nil(t, err)
	err = s.Create(ctx, model.Subscription{
		InterestId: "sub2",
		GroupId:    "group0",
		UserId:     "user1",
		Url:        "url0",
		Secret:     []byte{1, 2, 3},
		Format:     model.FormatCeJs,
	})
	require.Nil(t, err)
	//
	cases := map[string]struct {
		groupId string
		userId  string
		limit   uint32
		out     []model.Subscription
		err     error
	}{
		"empty": {
			limit: 10,
		},
		"1": {
			limit:   1,
			groupId: "group0",
			userId:  "user0",
			out: []model.Subscription{
				{
					InterestId: "sub0",
					Url:        "url0",
					Secret:     []byte{1, 2, 3},
					Format:     model.FormatCeJs,
				},
			},
		},
		"2": {
			limit:   10,
			groupId: "group0",
			userId:  "user0",
			out: []model.Subscription{
				{
					InterestId: "sub0",
					Url:        "url0",
					Secret:     []byte{1, 2, 3},
					Format:     model.FormatCeJs,
				},
				{
					InterestId: "sub1",
					Url:        "url0",
					Secret:     []byte{1, 2, 3},
					Format:     model.FormatCeJs,
				},
			},
		},
	}
	//
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			var out []model.Subscription
			out, err = s.ListByUser(ctx, c.limit, c.groupId, c.userId)
			assert.Equal(t, c.out, out)
			assert.ErrorIs(t, err, c.err)
		})
	}
}

func TestStorageMongo_ChangeOwner(t *testing.T) {
	//
	collName := fmt.Sprintf("subscriptions-test-%d", time.Now().UnixMicro())
	dbCfg := config.DbConfig{
		Uri:  dbUri,
		Name: "subscriptions",
	}
	dbCfg.Table.Name = collName
	dbCfg.Tls.Enabled = true
	dbCfg.Tls.Insecure = true
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	s, err := NewStorage(ctx, dbCfg)
	require.Nil(t, err)
	defer clear(ctx, t, s.(storageMongo))
	//
	err = s.Create(ctx, model.Subscription{
		InterestId: "sub0",
		GroupId:    "group0",
		UserId:     "user0",
		Url:        "url0",
	})
	require.Nil(t, err)
	err = s.Create(ctx, model.Subscription{
		InterestId: "sub1",
		GroupId:    "group0",
		UserId:     "user1",
		Url:        "url0",
	})
	require.Nil(t, err)
	err = s.Create(ctx, model.Subscription{
		InterestId: "sub2",
		GroupId:    "group0",
		UserId:     "user1",
		Url:        "url0",
	})
	require.Nil(t, err)
	//
	cases := map[string]struct {
		oldGroupId string
		oldUserId  string
		newGroupId string
		newUserId  string
		n          int64
		err        error
	}{
		"none": {
			oldGroupId: "group1",
			oldUserId:  "user1",
		},
		"one": {
			oldGroupId: "group0",
			oldUserId:  "user0",
			n:          1,
		},
		"two": {
			oldGroupId: "group0",
			oldUserId:  "user1",
			newGroupId: "group2",
			newUserId:  "user2",
			n:          2,
		},
	}
	//
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var n int64
			n, err = s.ChangeOwner(context.TODO(), c.oldGroupId, c.oldUserId, c.newGroupId, c.newUserId)
			assert.Equal(t, c.n, n)
			assert.ErrorIs(t, err, c.err)
		})
	}
}

func TestStorageMongo_ListAll(t *testing.T) {
	//
	collName := fmt.Sprintf("callbacks-test-%d", time.Now().UnixMicro())
	dbCfg := config.DbConfig{
		Uri:  dbUri,
		Name: "subscriptions",
	}
	dbCfg.Table.Name = collName
	dbCfg.Tls.Enabled = true
	dbCfg.Tls.Insecure = true
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()
	s, err := NewStorage(ctx, dbCfg)
	require.Nil(t, err)
	assert.NotNil(t, s)
	//
	defer clear(ctx, t, s.(storageMongo))
	//
	err = s.Create(ctx, model.Subscription{
		InterestId: "sub0",
		GroupId:    "group0",
		UserId:     "user0",
		Url:        "url0",
		Secret:     []byte{1, 2, 3},
		Format:     model.FormatCeJs,
	})
	require.Nil(t, err)
	err = s.Create(ctx, model.Subscription{
		InterestId: "sub1",
		GroupId:    "group0",
		UserId:     "user0",
		Url:        "url1",
		Secret:     []byte{1, 2, 3},
		Format:     model.FormatCeJs,
	})
	require.Nil(t, err)
	err = s.Create(ctx, model.Subscription{
		InterestId: "sub2",
		GroupId:    "group0",
		UserId:     "user1",
		Url:        "url0",
		Secret:     []byte{1, 2, 3},
		Format:     model.FormatCeJs,
	})
	require.Nil(t, err)
	//
	cases := map[string]struct {
		limit  uint32
		cursor string
		n      int
		err    error
	}{
		"ok": {
			limit: 10,
			n:     3,
		},
	}
	//
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			var out []model.Subscription
			out, err = s.ListAll(ctx, c.limit, c.cursor)
			assert.Equal(t, c.n, len(out))
			assert.ErrorIs(t, err, c.err)
		})
	}
}
