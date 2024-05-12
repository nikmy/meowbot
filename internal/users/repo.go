package users

import (
	"context"
	"github.com/nikmy/meowbot/internal/interviews"
	"github.com/nikmy/meowbot/pkg/logger"
	"slices"
	"sort"
	"time"

	"github.com/nikmy/meowbot/internal/repo"
	"github.com/nikmy/meowbot/pkg/errors"
)

type User struct {
	Assigned []Interview `json:"assigned" bson:"assigned"`
	Username string      `json:"username" bson:"username"`
	Employee bool        `json:"employee" bson:"employee"`
}

type Interview struct {
	ID       string   `json:"id"        bson:"id"`
	Role     Role     `json:"role"      bson:"role"`
	TimeSlot [2]int64 `json:"time_slot" bson:"time_slot"`
}

func getIntervals(interviews []Interview) [][2]int64 {
	intervals := make([][2]int64, 0, len(interviews))
	for _, i := range interviews {
		intervals = append(intervals, i.TimeSlot)
	}
	return intervals
}

type Role = interviews.Role

func (u User) Recipient() string {
	return u.Username
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
		return nil, nil
	}

	return &users[0], nil
}

var minLen = time.Hour.Milliseconds() // TODO: config

func (r *repoAPI) Match(ctx context.Context, targetInterval [2]int64) ([]User, error) {
	return r.repo.Select(
		ctx,
		repo.ByField("employee", true),
		repo.Where(func(u User) bool {
			return !intersect(getIntervals(u.Assigned), targetInterval, minLen)
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
	err := r.repo.Txn(ctx, func() error {
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
			func(u User) User {
				switch u.Username {
				case interviewer:
					interview.Role = interviews.RoleInterviewer
					u.Assigned = slices.Insert(u.Assigned, ivIdx, interview)
				case candidate:
					interview.Role = interviews.RoleCandidate
					u.Assigned = slices.Insert(u.Assigned, cdIdx, interview)
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

func addInterval(intervals [][2]int64, t [2]int64) (int, bool) {
	idx := sort.Search(len(intervals), func(i int) bool {
		return intervals[i][0] >= t[0]
	})

	return idx, idx == len(intervals) || intervals[idx][0] >= t[1]
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
