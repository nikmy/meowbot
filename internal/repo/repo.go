package repo

import (
	"context"
)

type Repo[T any] interface {
	Create(ctx context.Context, data T) (id string, err error)
	Select(ctx context.Context, filters... Filter) (selected []T, err error)
	Update(ctx context.Context, filter Filter, update func(T) T) (err error)
	Delete(ctx context.Context, id string) (err error)

	Close(ctx context.Context) error
}
