package users

import (
	"context"
	"slices"
	"sort"

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
			_, canInsert := addMeeting(u.Meetings, targetInterval)
			return canInsert
		}),
	)
}

func (r *repoAPI) Schedule(
	ctx context.Context,
	candidate string,
	interviewer string,
	meeting Meeting,
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

		ivIdx, scheduled := addMeeting(inter.Meetings, meeting)
		if !scheduled {
			return errors.Fail("schedule meeting for interviewer")
		}

		cdIdx, scheduled := addMeeting(cand.Meetings, meeting)
		if !scheduled {
			return errors.Fail("schedule meeting for candidate")
		}

		err = r.repo.Update(
			ctx,
			func(u *User) {
				switch u.Username {
				case interviewer:
					u.Meetings = slices.Insert(u.Meetings, ivIdx, meeting)
				case candidate:
					u.Meetings = slices.Insert(u.Meetings, cdIdx, meeting)
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
	return ok, errors.WrapFail(err, "schedule meeting to user")
}

func (r *repoAPI) Free(
	ctx context.Context,
	interviewer string,
	interval [2]int64,
) error {
	user, err := r.Get(ctx, interviewer)
	if err != nil {
		return errors.WrapFail(err, "get interviewer")
	}

	idx := sort.Search(len(user.Meetings), func(i int) bool {
		return user.Meetings[i][0] == interval[0]
	})
	if idx == len(user.Meetings) {
		return errors.Error("no intervals with specified start")
	}

	if user.Meetings[idx][1] != interval[1] {
		return errors.Error("no intervals with specified end")
	}

	updated := slices.Delete(user.Meetings, idx, idx+1)
	err = r.repo.Update(
		ctx,
		func(u *User) {
			u.Meetings = updated
		},
		repo.ByField("username", interviewer),
	)
	if err != nil {
		return errors.WrapFail(err, "update user intervals")
	}

	return nil
}
