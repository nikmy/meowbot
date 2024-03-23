package pubsub

import (
	"context"
	"encoding/json"
	"github.com/nikmy/meowbot/internal/repo"
	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/logger"
	"github.com/segmentio/kafka-go"
	"time"
)

func NewKafkaProducer(ctx context.Context, cfg Config, log logger.Logger) *kafkaProducer {
	c := &kafka.Client{
		Addr:    kafka.TCP(cfg.Brokers...),
		Timeout: time.Second * 5,
	}

	return &kafkaProducer{
		client: c,
		logger: log.With("kafka_producer"),
	}
}

type kafkaProducer struct {
	client  *kafka.Client
	logger  logger.Logger
}

func (p *kafkaProducer) Broadcast(ctx context.Context, channels []string, data repo.Reminder) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return errors.WrapFail(err, "marshal data to json")
	}

	for _, channel := range channels {
		record := kafka.Record{
			Key:   kafka.NewBytes([]byte(data.GetID())),
			Value: kafka.NewBytes(bytes),
		}

		p.client.Produce(ctx, &kafka.ProduceRequest{
			Topic:        channel,
			RequiredAcks: 1,
			Records:      kafka.NewRecordReader(record),
		})
	}

	return nil
}
