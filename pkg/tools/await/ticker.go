package await

import (
	"context"
	"reflect"
	"time"
)

type tickerAwaiter struct {
	*time.Ticker
}

func Tick(interval time.Duration) Awaiter {
	return &tickerAwaiter{time.NewTicker(interval)}
}

func (t *tickerAwaiter) Await(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-t.Ticker.C:
		return true
	}
}

func (t *tickerAwaiter) Value() (any, bool) {
	return struct{}{}, false
}

func (t *tickerAwaiter) bind() reflect.SelectCase {
	return reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(t.Ticker.C),
	}
}
