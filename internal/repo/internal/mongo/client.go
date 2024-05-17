package repo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/nikmy/meowbot/internal/repo/models"
	"github.com/nikmy/meowbot/pkg/errors"
)

type Config struct {
	Interval time.Duration `yaml:"interval"`

	URL     string        `yaml:"url"`
	Timeout time.Duration `yaml:"timeout"`

	Database string `yaml:"database"`

	Auth struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"auth"`

	Pool struct {
		MinSize uint64 `yaml:"minSize"`
		MaxSize uint64 `yaml:"maxSize"`
	}
}

func NewMongoClient(
	ctx context.Context,
	cfg Config,
	interviewsCollectionName string,
	usersCollectionName string,
) (*mongoClient, error) {
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
		users:      mongoUsers{db.Collection(string(usersCollectionName))},
		interviews: mongoInterviews{db.Collection(interviewsCollectionName)},
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
