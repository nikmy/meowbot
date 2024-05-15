package users

import (
	"context"
	"slices"

	"github.com/nikmy/meowbot/internal/interviews"
	"github.com/nikmy/meowbot/internal/repo"
	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/logger"
)

type UserDiff struct {
	Telegram *int64
	Username *string
	Employee *bool
}

func New(ctx context.Context, log logger.Logger, cfg repo.Config, src repo.DataSource) (API, error) {
	db, err := repo.New[User](ctx, cfg, src, log)
	if err != nil {
		return nil, errors.WrapFail(err, "init repo")
	}

	return &repoAPI{repo: db}, nil
}

type repoAPI struct {
	repo repo.Repo[User]
}

func (r *repoAPI) Upsert(ctx context.Context, username string, diff UserDiff) error {
	return r.repo.Update(
		ctx,
		func(u *User) {
			if diff.Employee != nil {
				u.Employee = *diff.Employee
			}
			if diff.Username != nil {
				u.Username = *diff.Username
			}
			if diff.Telegram != nil {
				u.Telegram = *diff.Telegram
			}
		},
		repo.ByField("username", username),
	)
}

func (r *repoAPI) Get(ctx context.Context, username string) (*User, error) {
	users, err := r.repo.Select(ctx, repo.ByField("username", username))
	if err != nil {
		return nil, errors.WrapFail(err, "select user by username")
	}

	if len(users) == 0 {
		return nil, nil
	}

	return &users[0], nil
}

func (r *repoAPI) Match(ctx context.Context, targetInterval [2]int64) ([]User, error) {
	return r.repo.Select(
		ctx,
		repo.ByField("employee", true),
		repo.Where(func(u User) bool {
			_, canInsert := addInterval(getIntervals(u.Assigned), targetInterval)
			return canInsert
		}),
	)
}

func (r *repoAPI) Assign(
	ctx context.Context,
	candidate string,
	interviewer string,
	interview Interview,
	onSuccess func() error,
) (bool, error) {
	ok, err := r.repo.Txn(ctx, func() error {
		inter, err := r.Get(ctx, interviewer)
		if err != nil {
			return errors.WrapFail(err, "get interviewer")
		}

		cand, err := r.Get(ctx, candidate)
		if err != nil {
			return errors.WrapFail(err, "get candidate")
		}

		ivIdx, assigned := addInterval(getIntervals(inter.Assigned), interview.TimeSlot)
		if !assigned {
			return errors.Fail("assign interval for interviewer")
		}

		cdIdx, assigned := addInterval(getIntervals(cand.Assigned), interview.TimeSlot)
		if !assigned {
			return errors.Fail("assign interval for candidate")
		}

		err = r.repo.Update(
			ctx,
			func(u *User) {
				switch u.Username {
				case interviewer:
					interview.Role = interviews.RoleInterviewer
					u.Assigned = slices.Insert(u.Assigned, ivIdx, interview)
				case candidate:
					interview.Role = interviews.RoleCandidate
					u.Assigned = slices.Insert(u.Assigned, cdIdx, interview)
				}
			},
			repo.Where(func(user User) bool {
				return user.Username == candidate || user.Username == interviewer
			}),
		)
		if err != nil {
			return errors.WrapFail(err, "update users intervals")
		}

		if onSuccess != nil {
			return onSuccess()
		}
		return nil
	})

	if err == nil && !ok {
		err = errors.Error("transaction aborted")
	}
	return ok, errors.WrapFail(err, "assign interval to user")
}

func (r *repoAPI) Free(
	ctx context.Context,
	interviewer User,
	interval [2]int64,
	onSuccess func() error,
) error {
	panic("TODO")
	//err := r.repo.Txn(ctx, func() error {
	//	user, err := r.Get(ctx, interviewer.Username)
	//	if err != nil {
	//		return errors.WrapFail(err, "get interviewer")
	//	}
	//
	//	idx := sort.Search(len(user.Intervals), func(i int) bool {
	//		return user.Intervals[i][0] == interval[0]
	//	})
	//	if idx == len(user.Intervals) {
	//		return errors.Error("no intervals with specified start")
	//	}
	//
	//	if user.Intervals[idx][1] != interval[1] {
	//		return errors.Error("no intervals with specified end")
	//	}
	//
	//	intervals := slices.Delete(user.Intervals, idx, idx+1)
	//
	//	err = r.repo.Update(
	//		ctx,
	//		func(u User) User {
	//			u.Intervals = intervals
	//			return u
	//		},
	//		repo.ByField("username", interviewer.Username),
	//	)
	//	if err != nil {
	//		return errors.WrapFail(err, "update user intervals")
	//	}
	//
	//	if onSuccess != nil {
	//		return onSuccess()
	//	}
	//	return nil
	//})
	//
	//return errors.WrapFail(err, "free interval for user")
}
