package interviews

import (
	"context"
	"encoding/base64"
	"encoding/hex"
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

func (r *repoAPI) Txn(ctx context.Context, do func() error) (bool, error) {
	return r.repo.Txn(ctx, do)
}

func (r *repoAPI) Schedule(ctx context.Context, id string, interviewer string, slot [2]int64) error {
	return r.repo.Update(
		ctx,
		func(i *Interview) {
			i.InterviewerTg = interviewer
			i.Interval = slot
		},
		repo.ByID(id),
	)
}

func (r *repoAPI) Create(ctx context.Context, vacancy string, candidateTg string) (string, error) {
	return r.repo.Insert(ctx, Interview{
		CandidateTg: candidateTg,
		Vacancy:     vacancy,
		Status:      StatusNew,
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

func (r *repoAPI) FindByUser(ctx context.Context, user string) ([]Interview, error) {
	cand, err := r.repo.Select(ctx, repo.ByField("candidate", user))
	if err != nil {
		return nil, err
	}

	inter, err := r.repo.Select(ctx, repo.ByField("interviewer", user))
	if err != nil {
		return nil, err
	}

	var oidBuf [12]byte

	all := append(cand, inter...)
	for i := range all {
		_, _ = hex.Decode(oidBuf[:], []byte(all[i].ID))
		all[i].ID = base64.StdEncoding.EncodeToString(oidBuf[:])
	}
	return all, nil
}

func (r *repoAPI) GetReadyAt(ctx context.Context, at int64) (interviews []Interview, err error) {
	return r.repo.Select(ctx, repo.Where(func(i Interview) bool {
		return i.Status == StatusScheduled && i.Interval[0] <= at && i.Interval[1] >= at
	}))
}

func (r *repoAPI) Cancel(ctx context.Context, id string, side Role) (err error) {
	return r.repo.Update(ctx, func(i *Interview) {
		i.Status = StatusCancelled
		i.Interval = [2]int64{}
		i.CancelledBy = side
		i.InterviewerTg = ""
	}, repo.ByID(id))
}

func (r *repoAPI) Done(ctx context.Context, id string) (err error) {
	return r.repo.Update(ctx, func(i *Interview) {
		i.Status = StatusFinished
	}, repo.ByID(id))
}

func (r *repoAPI) Close(ctx context.Context) error {
	return r.repo.Close(ctx)
}
