package telegram

import (
	"context"
	"time"

	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/repo/models"
	"github.com/nikmy/meowbot/pkg/logger"
)

func New(
	logger logger.Logger,
	cfg Config,
	interviews models.InterviewsRepo,
	users models.UsersRepo,
) (*Bot, error) {
	b, err := telebot.NewBot(telebot.Settings{
		Token:   cfg.Token,
		Updates: 256,
		Poller: &telebot.LongPoller{
			Timeout: cfg.PollInterval,
		},
	})
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		bot:        b,
		log:        logger,
		users:      users,
		interviews: interviews,
	}

	bot.applyNotifications(cfg)

	return bot, nil
}

type Bot struct {
	bot *telebot.Bot

	ctx context.Context
	log logger.Logger

	users      models.UsersRepo
	interviews models.InterviewsRepo

	notifyBefore []int64
	notifyPeriod time.Duration
}

func (b *Bot) Run(ctx context.Context) error {
	b.ctx = ctx
	b.setupHandlers()
	go b.bot.Start()
	b.runNotifier()
	return nil
}

func (b *Bot) Stop() {
	b.bot.Stop()
}
