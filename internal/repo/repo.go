package repo

import (
	"context"
)

type Repo[T any] interface {
	Txn(ctx context.Context, do func() error) error

	Create(ctx context.Context, data T) (id string, err error)
	Select(ctx context.Context, filters ...Filter) (selected []T, err error)
	Update(ctx context.Context, update func(T) T, filters ...Filter) (err error)
	Delete(ctx context.Context, id string) (err error)

	Close(ctx context.Context) error
}
