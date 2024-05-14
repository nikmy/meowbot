package repo

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/logger"
)

func New[T any](
	ctx context.Context,
	cfg Config,
	dataSource DataSource,
	log logger.Logger,
) (Repo[T], error) {
	if cfg.MongoCfg != nil {
		return newMongo[T](ctx, *cfg.MongoCfg, string(dataSource), log)
	}

	panic("unknown database")
}

func newMongo[T any](
	ctx context.Context,
	cfg MongoConfig,
	coll string,
	log logger.Logger,
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

	db := client.Database(cfg.Database)
	if db == nil {
		return nil, errors.WrapFail(err, "get database %s", cfg.Database)
	}

	collection := db.Collection(coll)
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

func (r *mongoRepo[T]) Insert(ctx context.Context, data T) (string, error) {
	var id string

	_, err := r.Txn(ctx, func() error {
		result, err := r.coll.InsertOne(ctx, data)
		if err != nil {
			return errors.WrapFail(err, "insert data")
		}

		id, err = r.makeID(result.InsertedID)
		return errors.WrapFail(err, "make object id")
	})

	return id, err
}

func (r *mongoRepo[T]) Select(ctx context.Context, filters ...Filter) ([]T, error) {
	f := r.applyFilters(filters...)
	mongoFilter, err := r.mongoFilter(f)
	if err != nil {
		return nil, errors.WrapFail(err, "make mongo filter")
	}

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

	return selected, errors.Join(errs...)
}

func (r *mongoRepo[T]) Update(ctx context.Context, update func(*T), filters ...Filter) error {
	mongoFilter, err := r.mongoFilter(r.applyFilters(filters...))
	if err != nil {
		return errors.WrapFail(err, "make mongo filter")
	}

	_, err = r.Txn(ctx, func() error {
		result := r.coll.FindOne(ctx, mongoFilter)

		err := result.Err()
		if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
			return errors.WrapFail(err, "find document to update")
		}

		var data T
		if err == nil {
			err = result.Decode(&data)
			if err != nil {
				return errors.WrapFail(err, "decode data")
			}
		}

		update(&data)
		opts := &options.UpdateOptions{Upsert: new(bool)}
		*opts.Upsert = true

		var req bson.M
		raw, err := json.Marshal(data)
		if err != nil {
			return errors.WrapFail(err, "marshal data")
		}
		err = json.Unmarshal(raw, &req)
		if err != nil {
			return errors.WrapFail(err, "unmarshal data to bson.M")
		}
		req = bson.M{"$set": req}

		_, err = r.coll.UpdateOne(ctx, mongoFilter, req, opts)
		return errors.WrapFail(err, "do upsert")
	})
	return errors.WrapFail(err, "perform update transaction")
}

func (r *mongoRepo[T]) Delete(ctx context.Context, id string) (bool, error) {
	f, err := r.oidFilter(id)
	if err != nil {
		return false, errors.WrapFail(err, "make id filter")
	}

	res, err := r.coll.DeleteOne(ctx, f)
	return res.DeletedCount == 1, errors.WrapFail(err, "delete data by oid")
}

func (r *mongoRepo[T]) Close(ctx context.Context) error {
	err := r.coll.Database().Client().Disconnect(ctx)
	return errors.WrapFail(err, "close mongo db connection")
}

func (r *mongoRepo[T]) makeID(id any) (string, error) {
	objID, ok := id.(primitive.ObjectID)
	if !ok {
		return "", errors.Error("bad id (%v) type", id)
	}

	b := ([12]byte)(objID)
	return base64.StdEncoding.EncodeToString(b[:]), nil
}

func (r *mongoRepo[T]) applyFilters(filters ...Filter) *filter {
	f := newFilter()
	for _, fn := range filters {
		fn(f)
	}
	return f
}

func (r *mongoRepo[T]) mongoFilter(f *filter) (bson.M, error) {
	mongoFilter := bson.M{}
	if f.id != nil {
		var err error
		mongoFilter, err = r.oidFilter(*f.id)
		if err != nil {
			return nil, errors.WrapFail(err, "make id filter")
		}
	}
	for field, val := range f.fields {
		mongoFilter[field] = val
	}
	return mongoFilter, nil
}

func (r *mongoRepo[T]) oidFilter(id string) (bson.M, error) {
	oid, err := base64.StdEncoding.DecodeString(id)
	if err != nil {
		return nil, errors.WrapFail(err, "decode id as base64")
	}

	var objectID [12]byte
	copy(objectID[:], oid)
	return bson.M{"_id": primitive.ObjectID(objectID)}, nil
}

func (r *mongoRepo[T]) Txn(ctx context.Context, do func() error) (bool, error) {
	session, err := r.coll.Database().Client().StartSession()
	if err != nil {
		return false, errors.WrapFail(err, "start mongo session")
	}

	ok := false
	err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		err := session.StartTransaction()
		if err != nil {
			return errors.WrapFail(err, "begin transaction")
		}

		err = do()
		if err != nil {
			r.log.Errorf("aborting txn because: %s", err.Error())
			err = session.AbortTransaction(sc)
			return errors.WrapFail(err, "abort transaction")
		}

		err = session.CommitTransaction(sc)
		ok = err == nil
		return errors.WrapFail(err, "commit transaction")
	})

	return ok, errors.WrapFail(err, "perform mongo transaction")
}
