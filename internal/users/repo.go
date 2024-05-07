package users

import (
	"context"
	"github.com/nikmy/meowbot/pkg/logger"
	"slices"
	"sort"
	"time"

	"github.com/nikmy/meowbot/internal/repo"
	"github.com/nikmy/meowbot/pkg/errors"
)

type User struct {
	Assigned []Interview // TODO: instead of Intervals

	Intervals [][2]int64 `json:"intervals" bson:"intervals"`
	Username  string     `json:"username" bson:"username"`
	Role      Role       `json:"role"     bson:"role"`
}

type Interview struct {
	ID string
	TimeSlot [2]int64
}

type Role int

const (
	Interviewer = Role(iota)
	Candidate
)

func (u User) Recipient() string {
	return "@" + u.Username
}

func New(ctx context.Context, log logger.Logger, cfg repo.Config) (API, error) {
	db, err := repo.New[User](ctx, cfg, log)
	if err != nil {
		return nil, errors.WrapFail(err, "init repo")
	}

	return &repoAPI{repo: db}, nil
}

type repoAPI struct {
	repo repo.Repo[User]
}

func (r *repoAPI) Add(ctx context.Context, user *User) error {
	if user == nil {
		return nil
	}

	_, err := r.repo.Create(ctx, *user)
	return err
}

func (r *repoAPI) Get(ctx context.Context, username string) (*User, error) {
	users, err := r.repo.Select(ctx, repo.ByField("username", username))
	if err != nil {
		return nil, errors.WrapFail(err, "select user by username")
	}

	if len(users) == 0 {
		return nil, errors.Error("no user with name %s", username)
	}

	return &users[0], nil
}

var minLen = time.Hour.Milliseconds() // TODO: config

func (r *repoAPI) Match(ctx context.Context, targetInterval [2]int64) ([]User, error) {
	return r.repo.Select(
		ctx,
		repo.ByField("role", Interviewer),
		repo.Where(func(u User) bool {
			return intersect(u.Intervals, targetInterval, minLen)
		}),
	)
}

func (r *repoAPI) Assign(
	ctx context.Context,
	candidate string,
	interviewer string,
	interval [2]int64,
	onSuccess func() error,
) (bool, error) {
	err := r.repo.Txn(ctx, func() error {
		inter, err := r.Get(ctx, interviewer)
		if err != nil {
			return errors.WrapFail(err, "get interviewer")
		}

		cand, err := r.Get(ctx, candidate)
		if err != nil {
			return errors.WrapFail(err, "get candidate")
		}

		var assigned bool
		inter.Intervals, assigned = addInterval(inter.Intervals, interval)
		if !assigned {
			return errors.Fail("assign interval for interviewer")
		}

		cand.Intervals, assigned = addInterval(cand.Intervals, interval)
		if !assigned {
			return errors.Fail("assign interval for candidate")
		}

		err = r.repo.Update(
			ctx,
			func(u User) User {
				switch u.Username {
				case interviewer:
					u.Intervals = inter.Intervals
				case candidate:
					u.Intervals = cand.Intervals
				}
				return u
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

	return err == nil, errors.WrapFail(err, "assign interval to user")
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

func addInterval(intervals [][2]int64, t [2]int64) ([][2]int64, bool) {
	idx := sort.Search(len(intervals), func(i int) bool {
		return intervals[i][0] >= t[0]
	})

	if idx < len(intervals) && intervals[idx][0] < t[1] {
		return intervals, false
	}

	intervals = slices.Insert(intervals, idx)
	return intervals, true
}

func intersect(intervals [][2]int64, t [2]int64, minIntersection int64) bool {
	for _, i := range intervals {
		// | | [ ]
		if i[1] <= t[0] {
			continue
		}

		// | | [ ]
		if t[1] <= i[0] {
			continue
		}

		// [ | ...
		if t[0] <= i[0] {
			// [ | <-> ] |
			if t[1] <= i[1] {
				return t[1]-i[0] >= minIntersection
			}

			// [ | <-> | ]
			return i[1]-i[0] >= minIntersection
		}

		// | [ ...

		// | [ ] |
		if t[1] <= i[1] {
			return true
		}

		// | [ <-> | ]
		return i[1]-t[0] >= minIntersection
	}

	return false
}
