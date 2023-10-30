package await

import (
	"context"
	"reflect"
)

func AllOf(waiters ...Awaiter) Awaiter {
	cases := make([]reflect.SelectCase, 0, len(waiters))
	for _, a := range waiters {
		cases = append(cases, a.bind())
	}
	return &allOfAwaiter{cases: cases}
}

type allOfAwaiter struct {
	cases []reflect.SelectCase
	rest  int
	all   []any
}

func (a *allOfAwaiter) Await(ctx context.Context) bool {
	a.rest = len(a.cases)
	a.cases = append(a.cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(ctx.Done()),
	})

	a.cases[0], a.cases[a.rest] = a.cases[a.rest], a.cases[0]
	defer func() {
		a.cases[0], a.cases[a.rest] = a.cases[a.rest], a.cases[0]
		a.cases = a.cases[:len(a.cases)]
	}()

	for a.rest > 0 {
		if !a.waitNext() {
			return false
		}
	}
	return true
}

func (a *allOfAwaiter) waitNext() bool {
	choice, val, _ := reflect.Select(a.cases)
	a.all = append(a.all, val.Interface())
	a.cases[choice], a.cases[a.rest] = a.cases[a.rest], a.cases[choice]
	a.rest--
	return choice != 0
}

func (a *allOfAwaiter) Value() (any, bool) {
	return a.all, len(a.all) != 0
}

func (a *allOfAwaiter) bind() reflect.SelectCase {
	panic("await: avoid combine combinators")
}
