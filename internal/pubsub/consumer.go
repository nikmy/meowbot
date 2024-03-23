package pubsub

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/logger"
)

func NewKafkaConsumer(ctx context.Context, cfg Config, log logger.Logger) (*kafkaConsumer, error) {
	readers := make(map[string]*kafka.Reader, len(cfg.Topics))

	c := kafka.Client{
		Addr:    kafka.TCP(cfg.Brokers...),
		Timeout: time.Second * 5,
	}

	now := time.Now().UnixMilli()

	topics := make(map[string][]kafka.OffsetRequest, len(cfg.Topics))
	for id, topicInfo := range cfg.Topics {
		for i := range topicInfo {
			topics[id] = append(topics[id], kafka.OffsetRequest{
				Partition: topicInfo[i],
				Timestamp: now,
			})
		}
	}

	offsets, err := c.ListOffsets(ctx, &kafka.ListOffsetsRequest{
		Topics:         topics,
		IsolationLevel: kafka.ReadCommitted,
	})

	if err != nil {
		return nil, errors.WrapFail(err, "list offsets")
	}

	for topic := range topics {
		readerCfg := kafka.ReaderConfig{
			Topic:          topic,
			StartOffset:    offsets.Topics[topic][0].LastOffset,
			Brokers:        cfg.Brokers,
			Dialer:         nil,
			QueueCapacity:  1024,
			IsolationLevel: kafka.ReadCommitted,
			MaxAttempts:    3,
			// Logger:                 nil,
			// ErrorLogger:            nil,
		}
		readers[topic] = kafka.NewReader(readerCfg)
	}

	return &kafkaConsumer{
		readers: readers,
		logger:  log.With("kafka_consumer"),
	}, nil
}

type kafkaConsumer struct {
	readers map[string]*kafka.Reader // TODO: LRU / pool
	logger  logger.Logger
}

func (c *kafkaConsumer) HandleEvents(
	ctx context.Context,
	channel string,
	consumeFunc func([]byte),
) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := c.readers[channel].FetchMessage(ctx)
				if err != nil {
					c.logger.Error(errors.WrapFail(err, "fetch message"))
					continue
				}

				consumeFunc(msg.Value)
				err = c.readers[channel].CommitMessages(ctx, msg)
				if err != nil {
					c.logger.Error(errors.WrapFail(err, "commit message"))
					continue
				}
			}
		}
	}()
}
