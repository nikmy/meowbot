package users

import (
	"context"
	"github.com/nikmy/meowbot/internal/repo"
	"github.com/nikmy/meowbot/pkg/errors"
)

type userModel struct {
	ID       int64  `json:"id"       bson:"user_id"`
	Username string `json:"username" bson:"username"`
}

type repoAPI struct {
	repo repo.Repo[userModel]
}

func (r *repoAPI) Add(ctx context.Context, user *User) error {
	if user == nil {
		return nil
	}

	u := userModel{
		ID: user.ID,
		Username: user.Username,
	}

	_, err := r.repo.Create(ctx, u)
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

	user := User{ID: users[0].ID, Username: username}
	return &user, nil
}
