package repo

import (
	"context"
	"math/rand"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/logger"
)

var (
	collectionIndex = mongo.IndexModel{
		Keys:    bson.D{{"remind_at", 1}},
		Options: options.Index().SetName("remind_time"),
	}
)

func newMongo(
	ctx context.Context,
	cfg MongoConfig,
	log logger.Logger,
) (*mongoRepo, error) {
	client, err := mongo.Connect(
		ctx,
		options.Client().
			ApplyURI(cfg.URL).
			SetTimeout(cfg.Timeout).
			SetAuth(options.Credential{
				Username: cfg.Auth.Username,
				Password: cfg.Auth.Password,
			}),
	)

	if err != nil {
		return nil, errors.WrapFail(err, "connect to mongo db")
	}

	collection := client.Database(cfg.Database).Collection(cfg.Collection)

	_, err = collection.Indexes().CreateOne(ctx, collectionIndex)
	if err != nil {
		return nil, errors.WrapFail(err, "create index")
	}

	return &mongoRepo{
		coll: collection,
		log:  log.With("mongo_repo"),
	}, nil
}

type mongoRepo struct {
	coll *mongo.Collection
	log  logger.Logger
}

func (m *mongoRepo) Get(ctx context.Context, id string) (Reminder, error) {
	result := m.coll.FindOne(ctx, bson.M{"id": id})
	if result.Err() != nil {
		return Reminder{}, errors.WrapFail(result.Err(), "find reminder")
	}

	var reminder Reminder

	err := result.Decode(&reminder)
	if err != nil {
		return Reminder{}, errors.WrapFail(err, "decode reminder")
	}

	raw, err := result.Raw()
	if err != nil {
		return Reminder{}, errors.WrapFail(err, "get raw bson")
	}

	reminder.Unique, _ = m.makeID(raw.Lookup().ObjectID())

	return reminder, nil
}

func (m *mongoRepo) GetReadyAt(ctx context.Context, at time.Time) ([]Reminder, error) {
	timeFilter := bson.D{
		{"remind_at", bson.D{
			{"$lte", at},
		}},
	}

	cur, err := m.coll.Find(ctx, timeFilter)
	if err != nil {
		return nil, errors.WrapFail(cur.Err(), "get ready reminders")
	}

	defer func() {
		err := cur.Close(ctx)
		if err != nil {
			m.log.Warn(errors.WrapFail(err, "close cursor"))
		}
	}()

	var (
		reminders []Reminder
		errs      []error
	)

	for cur.Next(ctx) {
		var r Reminder

		err := cur.Decode(&r)
		if err != nil {
			errs = append(errs, err)
		}
		r.Unique, _ = m.makeID(cur.Current.Lookup().ObjectID())
		reminders = append(reminders, r)
	}

	if cur.Err() != nil {
		return nil, errors.WrapFail(cur.Err(), "get ready reminders")
	}

	if len(errs) != 0 {
		err = errors.WrapFail(errors.Collapse(errs), "decode some reminders")
		m.log.Error(err)
	}

	return reminders, nil
}

func (m *mongoRepo) Create(ctx context.Context, data any, at time.Time, channels []string) (string, error) {
	reminder := Reminder{
		RandomID: rand.Uint32(),
		Channels: channels,
		RemindAt: at,
		Data:     data,
	}

	result, err := m.coll.InsertOne(ctx, reminder)
	if err != nil {
		return "", errors.WrapFail(err, "insert data")
	}

	id, err := m.makeID(result.InsertedID)
	return id, errors.WrapFail(err, "get object id")
}

func (m *mongoRepo) Delete(ctx context.Context, id string) (bool, error) {
	result, err := m.coll.DeleteOne(ctx, m.oidFilter(id))
	if err != nil {
		return false, errors.WrapFail(err, "delete reminder by oid")
	}

	return result.DeletedCount == 1, nil
}

func (m *mongoRepo) Update(ctx context.Context, id string, newData any, newAt time.Time) (bool, error) {
	update := bson.D{
		{"$set", bson.D{
			{"data", newData},
			{"remind_at", newAt},
		}},
	}

	result, err := m.coll.UpdateOne(ctx, m.oidFilter(id), update)
	if err != nil {
		return false, errors.WrapFail(err, "update data by oid")
	}

	return result.ModifiedCount == 1, nil
}

func (m *mongoRepo) Close(ctx context.Context) error {
	err := m.coll.Database().Client().Disconnect(ctx)
	return errors.WrapFail(err, "close mongo db connection")
}

func (m *mongoRepo) makeID(iid any) (string, error) {
	objID, ok := iid.(primitive.ObjectID)
	if !ok {
		return "", errors.Error("bad id type")
	}

	b := ([12]byte)(objID)
	return string(b[:]), nil
}

func (m *mongoRepo) oidFilter(oid string) bson.D {
	var objectID [12]byte
	copy(objectID[:], oid)
	return bson.D{{"_id", primitive.ObjectID(objectID)}}
}
