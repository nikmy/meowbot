package users

import (
	"context"
)

type API interface {
	Upsert(ctx context.Context, username string, user UserDiff) error
	Get(ctx context.Context, username string) (*User, error)

	Match(ctx context.Context, targetInterval [2]int64) ([]User, error)
	Assign(ctx context.Context, candidate, interviewer string, interview Interview, onSuccess func() error) (bool, error)
	Free(ctx context.Context, interviewer User, interval [2]int64, onSuccess func() error) error
}
