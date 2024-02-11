package repo

import (
	"context"
	"time"
)

type Repo interface {
	Get(ctx context.Context, id string) (data Reminder, err error)
	GetReadyAt(ctx context.Context, at time.Time) (data []Reminder, err error)

	Create(ctx context.Context, data any, at time.Time, channels []string) (id string, err error)
	Delete(ctx context.Context, id string) (deleted bool, err error)
	Update(ctx context.Context, id string, newData any, newAt *time.Time) (success bool, err error)

	Close(ctx context.Context) error
}

type Reminder struct {
	Unique    string   `json:"unique"         bson:"-"`
	Cancelled bool     `json:"cancelled"  bson:"cancelled"`
	Channels  []string `json:"channels"   bson:"channels"`

	RandomID uint32    `json:"random_id"  bson:"random_id"`
	RemindAt time.Time `json:"remind_at"  bson:"remind_at"`
	Data     any       `json:"data"       bson:"data"`
}
