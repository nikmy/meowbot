package pubsub

import "context"

type PubSub interface {
	Broadcast(ctx context.Context, channels []string, data any) error
	Subscribe(ctx context.Context, channel string) (string, error)
	Commit(ctx context.Context)

}
