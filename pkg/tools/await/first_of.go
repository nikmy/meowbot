package await

import (
	"context"
	"reflect"
)

func FirstOf(waiters ...Awaiter) Awaiter {
	cases := make([]reflect.SelectCase, 0, len(waiters))
	for _, a := range waiters {
		cases = append(cases, a.bind())
	}

	return &firstOfAwaiter{cases: cases}
}

type firstOfAwaiter struct {
	cases []reflect.SelectCase
	val   any
}

func (a *firstOfAwaiter) Await(ctx context.Context) (waited bool) {
	a.cases = append(a.cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(ctx.Done()),
	})

	choice, val, _ := reflect.Select(a.cases)
	a.val = val.Interface()

	a.cases = a.cases[:len(a.cases)-1]

	return choice != len(a.cases)
}

func (a *firstOfAwaiter) Value() (any, bool) {
	return a.val, a.val == nil
}

func (a *firstOfAwaiter) bind() reflect.SelectCase {
	panic("await: avoid combine combinators")
}
