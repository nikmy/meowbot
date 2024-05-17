package repo

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/nikmy/meowbot/internal/repo/models"
	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/logger"
)

func NewMongoClient(
	ctx context.Context,
	lof logger.Logger,
	cfg MongoConfig,
	interviewsSrc DataSource,
	usersSrc DataSource,
) (Client, error) {
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

	db := client.Database(cfg.Database, &options.DatabaseOptions{})
	return &mongoClient{
		c:          client,
		users:      mongoUsers{db.Collection(string(usersSrc))},
		interviews: mongoInterviews{db.Collection(string(interviewsSrc))},
	}, nil
}

type mongoClient struct {
	c          *mongo.Client
	users      mongoUsers
	interviews mongoInterviews
}

func (m *mongoClient) Interviews() models.InterviewsRepo {
	return m.interviews
}

func (m *mongoClient) Users() models.UsersRepo {
	return m.users
}

func (m *mongoClient) RunTxn(fn func(c Client)) {
	//TODO implement me
	panic("implement me")
}

