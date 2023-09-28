package bot

import (
	"context"

	"go.uber.org/zap"
	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/pkg/config"
)

func New(logger *zap.SugaredLogger, conf *config.TelegramConfig) (*Bot, error) {
	b, err := telebot.NewBot(telebot.Settings{
		URL:     "",
		Token:   conf.Token,
		Updates: 256,
		Poller: &telebot.LongPoller{
			Timeout: conf.PollInterval,
		},
	})
	return &Bot{b, logger}, err
}

type Bot struct {
	*telebot.Bot
	logger *zap.SugaredLogger
}

func (b *Bot) Run(ctx context.Context) error {
	go b.Start()
	return nil
}
