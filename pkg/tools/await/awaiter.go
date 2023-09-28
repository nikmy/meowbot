package await

import (
	"context"
	"reflect"
)

type Awaiter interface {
	Value() (any, bool)
	Await(ctx context.Context) (waited bool)
	bind() reflect.SelectCase
}

type noAwaiter struct{}

func (noAwaiter) Await(ctx context.Context) bool {
	return true
}

func (noAwaiter) Value() (any, bool) {
	return struct{}{}, false
}

func (n noAwaiter) bind() reflect.SelectCase {
	ch := make(chan struct{})
	close(ch)
	return reflect.SelectCase{
		Chan: reflect.ValueOf(ch),
		Dir:  reflect.SelectRecv,
	}
}
