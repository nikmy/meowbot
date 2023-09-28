package await

import (
	"context"
	"reflect"
	"sync"
	"time"
)

var timerPool = sync.Pool{
	New: newTimer,
}

func newTimer() any {
	return time.NewTimer(0)
}

type timerAwaiter struct {
	*time.Timer
}

func Until(ts time.Time, minWaitingTime time.Duration) Awaiter {
	waitTime := time.Until(ts)
	if waitTime < minWaitingTime {
		return noAwaiter{}
	}
	timer := timerPool.Get().(*time.Timer)
	timer.Reset(waitTime)
	return &timerAwaiter{timer}
}

func (t *timerAwaiter) Await(ctx context.Context) bool {
	defer func() {
		t.Stop()
		timerPool.Put(t.Timer)
	}()

	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func (t *timerAwaiter) Value() (any, bool) {
	return struct{}{}, false
}

func (t *timerAwaiter) bind() reflect.SelectCase {
	return reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(t.C),
	}
}
