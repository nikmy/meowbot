package interviews

import (
	"context"
	"github.com/nikmy/meowbot/internal/repo"
	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/logger"
	"go.mongodb.org/mongo-driver/mongo"
)

func New(ctx context.Context, log logger.Logger, cfg repo.MongoConfig) (API, error) {
	mongoRepo, err := repo.NewMongo[Interview](
		ctx,
		cfg,
		log,
		mongo.IndexModel{},
	)
	if err != nil {
		return nil, errors.WrapFail(err, "setup mongo")
	}

	return &repoAPI{repo: mongoRepo}, nil
}

type repoAPI struct {
	repo repo.Repo[Interview]
}

func (r *repoAPI) Schedule(ctx context.Context, cand string, inter string, slot [2]int64) error {
	return r.repo.Update(
		ctx,
		func(i Interview) Interview {
			i.Interval = slot
			return i
		},
		repo.ByField("candidate", cand),
		repo.ByField("interviewer", inter),
	)
}

func (r *repoAPI) Create(ctx context.Context, data any, interviewerTg string, candidateTg string) (string, error) {
	return r.repo.Create(ctx, Interview{
		InterviewerTg: interviewerTg,
		CandidateTg:   candidateTg,
		Data:          data,
		Status:        StatusNew,
	})
}

func (r *repoAPI) Delete(ctx context.Context, id string) error {
	return r.repo.Delete(ctx, id)
}

func (r *repoAPI) Find(ctx context.Context, id string) (bool, error) {
	found, err := r.repo.Select(ctx, repo.ByID(id))
	return len(found) == 1, err
}

func (r *repoAPI) GetReadyAt(ctx context.Context, at int64) (interviews []Interview, err error) {
	return r.repo.Select(ctx, repo.Where(func(i Interview) bool {
		return i.Status == StatusScheduled && i.Interval[0] <= at && i.Interval[1] >= at
	}))
}

func (r *repoAPI) Cancel(ctx context.Context, id string, reason string) (err error) {
	return r.repo.Update(ctx, func(i Interview) Interview {
		i.Status = StatusCancelled
		i.CancelReason = reason
		return i
	}, repo.ByID(id))
}

func (r *repoAPI) Done(ctx context.Context, id string) (err error) {
	return r.repo.Update(ctx, func(i Interview) Interview {
		i.Status = StatusFinished
		return i
	}, repo.ByID(id))
}

func (r *repoAPI) Close(ctx context.Context) error {
	return r.repo.Close(ctx)
}
