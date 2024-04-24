package users

import (
	"context"
	"slices"
	"sort"
	"time"

	"github.com/nikmy/meowbot/internal/repo"
	"github.com/nikmy/meowbot/pkg/errors"
)

type User struct {
	Intervals [][2]int64 `json:"intervals" bson:"intervals"`
	Username  string     `json:"username" bson:"username"`
	Role      Role       `json:"role"     bson:"role"`
}

type Role int

const (
	Interviewer = Role(iota)
	Candidate
)

func (u User) Recipient() string {
	return "@" + u.Username
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

func (r *repoAPI) Assign(ctx context.Context, interviewer User, interval [2]int64) (bool, error) {
	// FIXME TODO: transaction
	user, err := r.Get(ctx, interviewer.Username)
	if err != nil {
		return false, errors.WrapFail(err, "get interviewer")
	}

	intervals, assigned := addInterval(user.Intervals, interval)
	if !assigned {
		return false, nil
	}

	err = r.repo.Update(ctx, repo.ByField("username", interviewer.Username), func(u User) User {
		u.Intervals = intervals
		return u
	})
	return err == nil, errors.WrapFail(err, "update user intervals")
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
