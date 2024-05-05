package repo

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/logger"
)

func NewMongo[T any](
	ctx context.Context,
	cfg MongoConfig,
	log logger.Logger,
	collectionIndex mongo.IndexModel,
) (Repo[T], error) {
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

	return &mongoRepo[T]{
		coll: collection,
		log:  log.With("mongo_repo"),
	}, nil
}

type mongoRepo[T any] struct {
	coll *mongo.Collection
	acid mongo.Session
	log  logger.Logger
}

func (r *mongoRepo[T]) Create(ctx context.Context, data T) (string, error) {
	result, err := r.coll.InsertOne(ctx, data)
	if err != nil {
		return "", errors.WrapFail(err, "insert data")
	}

	id, err := r.makeID(result.InsertedID)
	return id, errors.WrapFail(err, "make object id")
}

func (r *mongoRepo[T]) Select(ctx context.Context, filters ...Filter) ([]T, error) {
	f := r.applyFilters(filters...)
	mongoFilter := r.mongoFilter(f)

	c, err := r.coll.Find(ctx, mongoFilter)
	if err != nil {
		return nil, errors.WrapFail(err, "find objects")
	}

	var (
		selected []T
		errs     []error
	)

	for c.Next(ctx) {
		var data T
		err = c.Decode(&data)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if f.fn == nil || (*f.fn)(data) {
			selected = append(selected, data)
		}
	}

	if c.Err() != nil {
		errs = append(errs, errors.WrapFail(err, "move cursor"))
	}

	return selected, errors.Join(errs)
}

func (r *mongoRepo[T]) Update(ctx context.Context, update func(T) T, filters ...Filter) error {
	mongoFilter := r.mongoFilter(r.applyFilters(filters...))

	result := r.coll.FindOne(ctx, mongoFilter)
	if err := result.Err(); err != nil {
		return errors.WrapFail(err, "find document to update")
	}

	var data T
	err := result.Decode(&data)
	if err != nil {
		return errors.WrapFail(err, "decode data")
	}

	data = update(data)
	opts := &options.UpdateOptions{Upsert: new(bool)}
	*opts.Upsert = true

	_, err = r.coll.UpdateOne(ctx, mongoFilter, data, opts)
	return errors.WrapFail(err, "do upsert")
}

func (r *mongoRepo[T]) Delete(ctx context.Context, id string) error {
	_, err := r.coll.DeleteOne(ctx, r.oidFilter(id))
	return errors.WrapFail(err, "delete data by oid")
}

func (r *mongoRepo[T]) Close(ctx context.Context) error {
	err := r.coll.Database().Client().Disconnect(ctx)
	return errors.WrapFail(err, "close mongo db connection")
}

func (r *mongoRepo[T]) makeID(iid any) (string, error) {
	objID, ok := iid.(primitive.ObjectID)
	if !ok {
		return "", errors.Error("bad id type")
	}

	b := ([12]byte)(objID)
	return string(b[:]), nil
}

func (r *mongoRepo[T]) applyFilters(filters ...Filter) *filter {
	f := &filter{}
	for _, fn := range filters {
		fn(f)
	}
	return f
}

func (r *mongoRepo[T]) mongoFilter(f *filter) bson.M {
	var mongoFilter bson.M
	if f.id != nil {
		mongoFilter = r.oidFilter(*f.id)
	}
	for field, val := range f.fields {
		mongoFilter[field] = val
	}
	return mongoFilter
}

func (r *mongoRepo[T]) oidFilter(oid string) bson.M {
	var objectID [12]byte
	copy(objectID[:], oid)
	return bson.M{"_id": primitive.ObjectID(objectID)}
}

func (r *mongoRepo[T]) Txn(ctx context.Context, do func() error) error {
	session, err := r.coll.Database().Client().StartSession()
	if err != nil {
		return errors.WrapFail(err, "start mongo session")
	}

	err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		err := session.StartTransaction()
		if err != nil {
			return errors.WrapFail(err, "begin transaction")
		}

		err = do()
		if err != nil {
			r.log.Infof("aborting txn because: %s", err.Error())
			err = session.AbortTransaction(sc)
			return errors.WrapFail(err, "abort transaction")
		}

		err = session.CommitTransaction(sc)
		return errors.WrapFail(err, "commit transaction")
	})

	return errors.WrapFail(err, "perform mongo transaction")
}
