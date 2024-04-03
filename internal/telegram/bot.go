package telegram

import (
	"context"

	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/repo"
	"github.com/nikmy/meowbot/pkg/logger"
)

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

	repo   repo.Repo
	logger logger.Logger
}

func (b *Bot) Run(ctx context.Context) error {
	b.Handle(telebot.OnCallback, b.handleClick)
	go b.Start()
	return nil
}

func (b *Bot) BindInline(button *telebot.InlineButton) error {
	return nil
}

func (b *Bot) handleClick(c telebot.Context) error {
	// TODO: routing
	return nil
}
