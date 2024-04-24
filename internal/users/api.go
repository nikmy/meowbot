package users

import (
	"context"
)

type API interface {
	Add(ctx context.Context, user *User) error
	Get(ctx context.Context, username string) (*User, error)

	// TODO: Assign is part of Match

	Match(ctx context.Context, targetInterval [2]int64) ([]User, error)
	Assign(ctx context.Context, interviewer User, interval [2]int64) (bool, error)
}
