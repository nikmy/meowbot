package repo

import (
	"context"

	mongorepo "github.com/nikmy/meowbot/internal/repo/internal/mongo"
	"github.com/nikmy/meowbot/internal/repo/models"
	"github.com/nikmy/meowbot/internal/repo/txn"
)

type Client interface {
	Interviews() models.InterviewsRepo
	Users() models.UsersRepo
	Make(lvl txn.Model) txn.Txn
}

type MongoConfig = mongorepo.Config

func NewMongoClient(
	ctx context.Context,
	cfg mongorepo.Config,
	interviewsSrc string,
	usersSrc string,
) (Client, error) {
	return mongorepo.NewMongoClient(ctx, cfg, interviewsSrc, usersSrc)
}
