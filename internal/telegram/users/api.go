package users

import (
	"context"
	tb "gopkg.in/telebot.v3"
)

type User = tb.User

type API interface {
	Add(ctx context.Context, user *User) error
	Get(ctx context.Context, username string) (*User, error)
}
