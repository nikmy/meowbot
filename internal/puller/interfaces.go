package puller

import (
	"context"
	"github.com/nikmy/meowbot/internal/repo"
	"time"
)

type Puller interface {
	DoWork(ctx context.Context) error
}

type broadcaster interface {
	Broadcast(ctx context.Context, channels []string, data repo.Reminder) error
}

type db interface {
	GetReadyAt(ctx context.Context, at time.Time) ([]repo.Reminder, error)
}
