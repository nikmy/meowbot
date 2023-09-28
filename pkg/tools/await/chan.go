package await

import (
	"context"
	"reflect"
)

func FromChan[T any](ch chan T) Awaiter {
	return &chanAwaiter[T]{
		ch: ch,
	}
}

func ToChan[T any](ch chan T, value T) Awaiter {
	return &chanAwaiter[T]{
		send: true,
		ch:   ch,
		val:  value,
	}
}

type chanAwaiter[T any] struct {
	send bool
	val  T
	ch   chan T
}

func (a *chanAwaiter[T]) Await(ctx context.Context) (waited bool) {
	select {
	case <-ctx.Done():
		return false
	case a.val = <-a.ch:
		return true
	}
}

func (a *chanAwaiter[T]) Value() (any, bool) {
	return a.val, !a.send
}

func (a *chanAwaiter[T]) bind() reflect.SelectCase {
	c := reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(a.ch),
	}
	if a.send {
		c.Dir = reflect.SelectSend
	}
	return c
}
