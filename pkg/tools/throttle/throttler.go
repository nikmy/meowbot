package throttle

import (
	"context"
	"github.com/nikmy/meowbot/pkg/tools/await"
	"time"
)

func New(interval time.Duration, capacity int) *throttler {
	return &throttler{
		todo:     make(chan func(), capacity),
		interval: interval,
	}
}

type throttler struct {
	todo     chan func()
	interval time.Duration
}

func (t *throttler) Run(ctx context.Context) error {
	a := await.AllOf(
		await.Tick(t.interval),
		await.FromChan(t.todo),
	)

	go func() {
		for {
			if !a.Await(ctx) {
				return
			}
			v, _ := a.Value()
			v.([]func())[0]()
		}
	}()
	return nil
}

func (t *throttler) Stop() {
	close(t.todo)
}

func (t *throttler) Do(ctx context.Context, action func()) bool {
	return await.ToChan(t.todo, action).Await(ctx)
}
