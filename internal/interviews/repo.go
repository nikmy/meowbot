package interviews

import (
	"context"

	"github.com/nikmy/meowbot/internal/repo"
)

type repoAPI struct {
	repo repo.Repo[Interview]
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

func (r *repoAPI) Propose(ctx context.Context, id string, intervals [][2]int64) error {
	return r.repo.Update(ctx, repo.ByID(id), func(i Interview) Interview {
		i.Intervals = intervals
		i.Status = StatusAsk
		return i
	})
}

func (r *repoAPI) Accept(ctx context.Context, id string, interval [2]int64) error {
	return r.repo.Update(ctx, repo.ByID(id), func(i Interview) Interview {
		i.Intervals = [][2]int64{interval}
		i.Status = StatusAccepted
		return i
	})
}

func (r *repoAPI) Decline(ctx context.Context, id string) error {
	return r.repo.Update(ctx, repo.ByID(id), func(i Interview) Interview {
		i.Intervals = nil
		i.Status = StatusDeclined
		return i
	})
}

func (r *repoAPI) GetReadyAt(ctx context.Context, at int64) (interviews []Interview, err error) {
	return r.repo.Select(ctx, repo.Where(func(i Interview) bool {
		return i.Status == StatusAccepted && i.Intervals[0][0] <= at && i.Intervals[0][1] >= at
	}))
}

func (r *repoAPI) Cancel(ctx context.Context, id string, reason string) (err error) {
	return r.repo.Update(ctx, repo.ByID(id), func(i Interview) Interview {
		i.Intervals = nil
		i.Status = StatusCancelled
		i.CancelReason = reason
		return i
	})
}

func (r *repoAPI) Done(ctx context.Context, id string) (err error) {
	return r.repo.Update(ctx, repo.ByID(id), func(i Interview) Interview {
		i.Intervals = nil
		i.Status = StatusFinished
		return i
	})
}

func (r *repoAPI) Close(ctx context.Context) error {
	return r.repo.Close(ctx)
}
