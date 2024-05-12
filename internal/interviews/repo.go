package interviews

import (
	"context"
	"github.com/nikmy/meowbot/internal/repo"
	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/logger"
)

func New(ctx context.Context, log logger.Logger, cfg repo.Config, src repo.DataSource) (API, error) {
	mongoRepo, err := repo.New[Interview](ctx, cfg, src, log)
	if err != nil {
		return nil, errors.WrapFail(err, "setup mongo")
	}

	return &repoAPI{repo: mongoRepo}, nil
}

type repoAPI struct {
	repo repo.Repo[Interview]
}

func (r *repoAPI) Schedule(ctx context.Context, id string, interviewer string, slot [2]int64) error {
	return r.repo.Update(
		ctx,
		func(i Interview) Interview {
			i.InterviewerTg = interviewer
			i.Interval = slot
			return i
		},
		repo.ByID(id),
	)
}

func (r *repoAPI) Create(ctx context.Context, data []byte, candidateTg string) (string, error) {
	return r.repo.Create(ctx, Interview{
		CandidateTg:   candidateTg,
		Data:          data,
		Status:        StatusNew,
	})
}

func (r *repoAPI) Delete(ctx context.Context, id string) (bool, error) {
	return r.repo.Delete(ctx, id)
}

func (r *repoAPI) Find(ctx context.Context, id string) (*Interview, error) {
	found, err := r.repo.Select(ctx, repo.ByID(id))
	if len(found) == 0 {
		return nil, err
	}

	i := found[0]
	return &i, err
}

func (r *repoAPI) FindByCandidate(ctx context.Context, candidate string) ([]Interview, error) {
	return r.repo.Select(ctx, repo.ByField("candidate", candidate))
}

func (r *repoAPI) GetReadyAt(ctx context.Context, at int64) (interviews []Interview, err error) {
	return r.repo.Select(ctx, repo.Where(func(i Interview) bool {
		return i.Status == StatusScheduled && i.Interval[0] <= at && i.Interval[1] >= at
	}))
}

func (r *repoAPI) Cancel(ctx context.Context, id string, side Role) (err error) {
	return r.repo.Update(ctx, func(i Interview) Interview {
		i.Status = StatusCancelled
		i.CancelledBy = side
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
