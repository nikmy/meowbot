package telegram

import (
	"context"

	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/interviews"
	"github.com/nikmy/meowbot/internal/users"
	"github.com/nikmy/meowbot/pkg/logger"
)

const USAGE = ""

func New(logger logger.Logger, conf *Config) (*Bot, error) {
	b, err := telebot.NewBot(telebot.Settings{
		URL:     "",
		Token:   conf.Token,
		Updates: 256,
		Poller: &telebot.LongPoller{
			Timeout: conf.PollInterval,
		},
	})
	return &Bot{
		Bot:    b,
		logger: logger,
	}, err
}

type Bot struct {
	*telebot.Bot

	ctx        context.Context
	users      users.API
	interviews interviews.API
	logger     logger.Logger
}

func (b *Bot) Run(ctx context.Context) error {
	b.ctx = ctx
	b.setupHandlers()
	go b.Bot.Start()
	return nil
}
