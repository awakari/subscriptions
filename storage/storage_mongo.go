package storage

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/awakari/subscriptions/config"
	"github.com/awakari/subscriptions/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type storageMongo struct {
	conn *mongo.Client
	db   *mongo.Database
	coll *mongo.Collection
}

type recSub struct {
	Id           string        `bson:"_id,omitempty"`
	InterestId   string        `bson:"interestId"`
	GroupId      string        `bson:"groupId"`
	UserId       string        `bson:"userId"`
	Url          string        `bson:"url"`
	Secret       []byte        `bson:"secret,omitempty"`
	Fmt          int           `bson:"fmt"`
	IntervalMin  time.Duration `bson:"interval,omitempty"`
	LastResultAt time.Time     `bson:"last,omitempty"`
	ErrorCount   uint32        `bson:"errorCount,omitempty"`
}

const attrId = "_id"
const attrInterestId = "interestId"
const attrGroupId = "groupId"
const attrUserId = "userId"
const attrUrl = "url"
const attrSecret = "secret"
const attrFmt = "fmt"
const attrInterval = "interval"
const attrLast = "last"
const attrErrorCount = "errorCount"
const attrDeletedAt = "deletedAt"

var optsSrvApi = options.ServerAPI(options.ServerAPIVersion1)
var optsRead = options.
	FindOne().
	SetShowRecordID(false).
	SetProjection(projRead)
var optsReadPage = options.
	Find().
	SetShowRecordID(false).
	SetProjection(projRead).
	SetSort(projReadPageSort)
var optsReadPageByUrl = options.
	Find().
	SetShowRecordID(false).
	SetProjection(projReadPageByUrl).
	SetSort(projReadPageSort)
var optsReadAllPage = options.
	Find().
	SetShowRecordID(true).
	SetProjection(projReadAll).
	SetSort(projReadAllPageSort)
var projRead = bson.D{
	{
		Key:   attrInterestId,
		Value: 1,
	},
	{
		Key:   attrGroupId,
		Value: 1,
	},
	{
		Key:   attrUserId,
		Value: 1,
	},
	{
		Key:   attrUrl,
		Value: 1,
	},
	{
		Key:   attrSecret,
		Value: 1,
	},
	{
		Key:   attrFmt,
		Value: 1,
	},
	{
		Key:   attrInterval,
		Value: 1,
	},
	{
		Key:   attrLast,
		Value: 1,
	},
	{
		Key:   attrErrorCount,
		Value: 1,
	},
}
var projReadAll = bson.D{
	{
		Key:   attrId,
		Value: 1,
	},
	{
		Key:   attrInterestId,
		Value: 1,
	},
	{
		Key:   attrGroupId,
		Value: 1,
	},
	{
		Key:   attrUserId,
		Value: 1,
	},
	{
		Key:   attrUrl,
		Value: 1,
	},
}
var projReadPageByUrl = bson.D{
	{
		Key:   attrInterestId,
		Value: 1,
	},
}
var projReadPageSort = bson.D{
	{
		Key:   attrInterestId,
		Value: 1,
	},
	{
		Key:   attrUrl,
		Value: 1,
	},
}
var projReadAllPageSort = bson.D{
	{
		Key:   attrId,
		Value: 1,
	},
}
var indices = []mongo.IndexModel{
	{
		Keys: bson.D{
			{
				Key:   attrInterestId,
				Value: 1,
			},
			{
				Key:   attrUrl,
				Value: 1,
			},
		},
		Options: options.
			Index().
			SetUnique(true),
	},
	{
		Keys: bson.D{
			{
				Key:   attrInterestId,
				Value: 1,
			},
			{
				Key:   attrDeletedAt,
				Value: 1,
			},
		},
		Options: options.
			Index().
			SetUnique(false),
	},
	{
		Keys: bson.D{
			{
				Key:   attrGroupId,
				Value: 1,
			},
			{
				Key:   attrUserId,
				Value: 1,
			},
		},
		Options: options.
			Index().
			SetUnique(false),
	},
}

func NewStorage(ctx context.Context, cfgDb config.DbConfig) (s Storage, err error) {
	clientOpts := options.
		Client().
		ApplyURI(cfgDb.Uri).
		SetServerAPIOptions(optsSrvApi)
	if cfgDb.Tls.Enabled {
		clientOpts = clientOpts.SetTLSConfig(&tls.Config{InsecureSkipVerify: cfgDb.Tls.Insecure})
	}
	if len(cfgDb.UserName) > 0 {
		auth := options.Credential{
			Username:    cfgDb.UserName,
			Password:    cfgDb.Password,
			PasswordSet: len(cfgDb.Password) > 0,
		}
		clientOpts = clientOpts.SetAuth(auth)
	}
	conn, err := mongo.Connect(ctx, clientOpts)
	var stor storageMongo
	if err == nil {
		db := conn.Database(cfgDb.Name)
		coll := db.Collection(cfgDb.Table.Name)
		stor.conn = conn
		stor.db = db
		stor.coll = coll
		_, err = stor.ensureIndices(ctx, cfgDb)
	}
	if err == nil && cfgDb.Table.Shard {
		err = stor.shardCollection(ctx)
	}
	if err == nil {
		s = stor
	}
	if err != nil {
		err = fmt.Errorf("%w: %s", ErrInternal, err)
	}
	return
}

func (s storageMongo) ensureIndices(ctx context.Context, cfgDb config.DbConfig) ([]string, error) {
	retentionSeconds := int32(cfgDb.Table.Retention.Seconds())
	if retentionSeconds > 0 {
		indices = append(indices, mongo.IndexModel{
			Keys: bson.D{
				{
					Key:   attrDeletedAt,
					Value: 1,
				},
			},
			Options: options.
				Index().
				SetExpireAfterSeconds(retentionSeconds),
		})
	}
	return s.coll.Indexes().CreateMany(ctx, indices)
}

func (s storageMongo) shardCollection(ctx context.Context) (err error) {
	adminDb := s.conn.Database("admin")
	cmd := bson.D{
		{
			Key:   "shardCollection",
			Value: fmt.Sprintf("%s.%s", s.db.Name(), s.coll.Name()),
		},
		{
			Key: "key",
			Value: bson.D{
				{
					Key:   attrInterestId,
					Value: "hashed",
				},
			},
		},
	}
	err = adminDb.RunCommand(ctx, cmd).Err()
	return
}

func (s storageMongo) Close() error {
	return s.conn.Disconnect(context.TODO())
}

func (s storageMongo) Create(ctx context.Context, sub model.Subscription) (err error) {
	rec := recSub{
		InterestId:  sub.InterestId,
		GroupId:     sub.GroupId,
		UserId:      sub.UserId,
		Url:         sub.Url,
		Secret:      sub.Secret,
		Fmt:         int(sub.Format),
		IntervalMin: sub.IntervalMin,
	}
	_, err = s.coll.InsertOne(ctx, rec)
	if mongo.IsDuplicateKeyError(err) {
		// attempt to delete the tombstone and retry the creation
		r, errDel := s.coll.DeleteOne(ctx, bson.M{
			attrInterestId: sub.InterestId,
			attrUrl:        sub.Url,
			attrDeletedAt: bson.M{
				"$exists": true,
			},
		})
		if errDel == nil && r.DeletedCount > 0 {
			err = s.Create(ctx, sub)
		}
	}
	err = decodeError(err, sub.InterestId, sub.Url)
	return
}

func (s storageMongo) Read(ctx context.Context, interestId, groupId, userId, url string) (sub model.Subscription, err error) {
	q := bson.M{
		attrInterestId: interestId,
		attrUrl:        url,
		"$or": []bson.M{
			{
				"$and": []bson.M{
					{
						attrGroupId: groupId,
					},
					{
						attrUserId: userId,
					},
					{
						attrDeletedAt: bson.M{
							"$exists": false,
						},
					},
				},
			},
			{
				// support legacy records where group and user are unknown
				"$and": []bson.M{
					{
						attrGroupId: bson.M{
							"$exists": false,
						},
					},
					{
						attrUserId: bson.M{
							"$exists": false,
						},
					},
					{
						attrDeletedAt: bson.M{
							"$exists": false,
						},
					},
				},
			},
		},
	}
	var result *mongo.SingleResult
	result = s.coll.FindOne(ctx, q, optsRead)
	err = result.Err()
	var rec recSub
	if err == nil {
		err = result.Decode(&rec)
	}
	if err == nil {
		sub.GroupId = rec.GroupId
		sub.UserId = rec.UserId
		sub.Url = rec.Url
		sub.Secret = rec.Secret
		sub.Format = model.Format(rec.Fmt)
		sub.IntervalMin = rec.IntervalMin
		sub.LastResultAt = rec.LastResultAt
		sub.ErrorCount = rec.ErrorCount
	}
	err = decodeError(err, interestId, url)
	return
}

func (s storageMongo) Update(ctx context.Context, sub model.Subscription, deliveryFailed bool) error {
	q := bson.M{
		attrInterestId: sub.InterestId,
		attrUrl:        sub.Url,
		"$or": []bson.M{
			{
				"$and": []bson.M{
					{
						attrGroupId: sub.GroupId,
					},
					{
						attrUserId: sub.UserId,
					},
					{
						attrDeletedAt: bson.M{
							"$exists": false,
						},
					},
				},
			},
			{
				// support legacy records where group and user are unknown
				"$and": []bson.M{
					{
						attrGroupId: bson.M{
							"$exists": false,
						},
					},
					{
						attrUserId: bson.M{
							"$exists": false,
						},
					},
					{
						attrDeletedAt: bson.M{
							"$exists": false,
						},
					},
				},
			},
		},
	}
	u := bson.M{}
	switch deliveryFailed {
	case true:
		u["$inc"] = bson.M{
			attrErrorCount: 1,
		}
	default:
		u["$set"] = bson.M{
			attrErrorCount: 0,
			attrLast:       sub.LastResultAt,
		}
	}
	result, err := s.coll.UpdateOne(ctx, q, u)
	if err != nil {
		return decodeError(err, sub.InterestId, sub.Url)
	}
	if result.MatchedCount < 1 {
		return fmt.Errorf("%w: %s, %s", ErrNotFound, sub.InterestId, sub.Url)
	}
	return nil
}

func (s storageMongo) Delete(ctx context.Context, interestId, groupId, userId, url string) (err error) {
	q := bson.M{
		attrInterestId: interestId,
		attrUrl:        url,
		"$or": []bson.M{
			{
				"$and": []bson.M{
					{
						attrGroupId: groupId,
					},
					{
						attrUserId: userId,
					},
					{
						attrDeletedAt: bson.M{
							"$exists": false,
						},
					},
				},
			},
			{
				// support legacy records where group and user are unknown
				"$and": []bson.M{
					{
						attrGroupId: bson.M{
							"$exists": false,
						},
					},
					{
						attrUserId: bson.M{
							"$exists": false,
						},
					},
					{
						attrDeletedAt: bson.M{
							"$exists": false,
						},
					},
				},
			},
		},
	}
	u := bson.M{
		attrDeletedAt: time.Now().UTC(),
	}
	result, err := s.coll.UpdateOne(ctx, q, bson.M{
		"$set": u,
	})
	if err != nil {
		return decodeError(err, interestId, url)
	}
	if result.MatchedCount < 1 {
		return fmt.Errorf("%w: %s, %s", ErrNotFound, interestId, url)
	}
	return
}

func (s storageMongo) CountByInterest(ctx context.Context, interestId string) (count int64, err error) {
	count, err = s.coll.CountDocuments(ctx, bson.M{
		attrInterestId: interestId,
		attrDeletedAt: bson.M{
			"$exists": false,
		},
	})
	err = decodeError(err, interestId, "")
	return
}

func (s storageMongo) ListByInterest(ctx context.Context, limit uint32, interestId, cursor string) (page []model.Subscription, err error) {
	q := bson.M{
		attrInterestId: interestId,
		attrUrl: bson.M{
			"$gt": cursor,
		},
		attrDeletedAt: bson.M{
			"$exists": false,
		},
	}
	var cur *mongo.Cursor
	cur, err = s.coll.Find(ctx, q, optsReadPage.SetLimit(int64(limit)))
	var recs []recSub
	if err == nil {
		err = cur.All(ctx, &recs)
	}
	if err == nil {
		for _, rec := range recs {
			page = append(page, model.Subscription{
				GroupId:      rec.GroupId,
				UserId:       rec.UserId,
				Url:          rec.Url,
				Secret:       rec.Secret,
				Format:       model.Format(rec.Fmt),
				IntervalMin:  rec.IntervalMin,
				LastResultAt: rec.LastResultAt,
				ErrorCount:   rec.ErrorCount,
			})
		}
	}
	err = decodeError(err, interestId, cursor)
	return
}

func (s storageMongo) ListByUrl(ctx context.Context, limit uint32, url, cursor string) (page []string, err error) {
	q := bson.M{
		attrUrl: bson.M{
			"$regex": "^" + url, // by prefix to support both new urls with appended user id and legacy ones
		},
		attrInterestId: bson.M{
			"$gt": cursor,
		},
		attrDeletedAt: bson.M{
			"$exists": false,
		},
	}
	var cur *mongo.Cursor
	cur, err = s.coll.Find(ctx, q, optsReadPageByUrl.SetLimit(int64(limit)))
	var recs []recSub
	if err == nil {
		err = cur.All(ctx, &recs)
	}
	if err == nil {
		for _, rec := range recs {
			page = append(page, rec.InterestId)
		}
	}
	err = decodeError(err, cursor, url)
	return
}

func (s storageMongo) ListByUser(ctx context.Context, limit uint32, groupId, userId string) (page []model.Subscription, err error) {
	q := bson.M{
		attrGroupId: groupId,
		attrUserId:  userId,
		attrDeletedAt: bson.M{
			"$exists": false,
		},
	}
	var cur *mongo.Cursor
	cur, err = s.coll.Find(ctx, q, optsReadPage.SetLimit(int64(limit)))
	var recs []recSub
	if err == nil {
		err = cur.All(ctx, &recs)
	}
	if err == nil {
		for _, rec := range recs {
			page = append(page, model.Subscription{
				InterestId:   rec.InterestId,
				Url:          rec.Url,
				Secret:       rec.Secret,
				Format:       model.Format(rec.Fmt),
				IntervalMin:  rec.IntervalMin,
				LastResultAt: rec.LastResultAt,
			})
		}
	}
	err = decodeError(err, "", "")
	return
}

func (s storageMongo) CountAll(ctx context.Context) (count int64, err error) {
	count, err = s.coll.CountDocuments(ctx, bson.M{
		attrDeletedAt: bson.M{
			"$exists": false,
		},
	})
	err = decodeError(err, "", "")
	return
}

func (s storageMongo) ListAll(ctx context.Context, limit uint32, cursor string) (page []model.Subscription, err error) {
	var cursorObjId primitive.ObjectID
	switch cursor {
	case "":
		cursorObjId = primitive.NilObjectID
	default:
		cursorObjId, err = primitive.ObjectIDFromHex(cursor)
	}
	if err != nil {
		err = decodeError(err, cursor, "")
	}
	q := bson.M{
		attrId: bson.M{
			"$gt": cursorObjId,
		},
		attrDeletedAt: bson.M{
			"$exists": false,
		},
	}
	var cur *mongo.Cursor
	cur, err = s.coll.Find(ctx, q, optsReadAllPage.SetLimit(int64(limit)))
	var recs []recSub
	if err == nil {
		err = cur.All(ctx, &recs)
	}
	if err == nil {
		for _, rec := range recs {
			page = append(page, model.Subscription{
				InterestId: rec.InterestId,
				Url:        rec.Url,
				GroupId:    rec.GroupId,
				UserId:     rec.UserId,
				InternalId: rec.Id,
			})
		}
	}
	err = decodeError(err, "", "")
	return
}

func (s storageMongo) ChangeOwner(ctx context.Context, oldGroupId, oldUserId, newGroupId, newUserId string) (n int64, err error) {
	q := bson.M{
		attrGroupId: oldGroupId,
		attrUserId:  oldUserId,
		attrDeletedAt: bson.M{
			"$exists": false,
		},
	}
	u := bson.M{
		"$set": bson.M{
			attrGroupId: newGroupId,
			attrUserId:  newUserId,
		},
	}
	var result *mongo.UpdateResult
	result, err = s.coll.UpdateMany(ctx, q, u)
	switch {
	case err == nil:
		n = result.ModifiedCount
	default:
		err = fmt.Errorf("%w: failed to change owner: %s", ErrInternal, err)
	}
	return
}

func decodeError(src error, interestId, url string) (dst error) {
	switch {
	case src == nil:
	case errors.Is(src, mongo.ErrNoDocuments):
		dst = fmt.Errorf("%w: %s, %s", ErrNotFound, interestId, url)
	case mongo.IsDuplicateKeyError(src):
		dst = fmt.Errorf("%w: %s, %s", ErrConflict, interestId, url)
	default:
		dst = fmt.Errorf("%w: %s, %s, %s", ErrInternal, interestId, url, src)
	}
	return
}
