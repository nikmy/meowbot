package pubsub

import (
	"context"
)

func New() {
	var c kafka.Conn
}

type kafkaClient struct{}

func (c *kafkaClient) Broadcast(ctx context.Context, channels []string, data any) error {
	//TODO implement me
	panic("implement me")
}

func (c *kafkaClient) Subscribe(ctx context.Context, channel string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (c *kafkaClient) Commit(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}
