package telegram

import (
	"context"
	"time"

	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/repo"
	"github.com/nikmy/meowbot/internal/repo/txn"
	"github.com/nikmy/meowbot/pkg/logger"
)

func New(logger logger.Logger, cfg Config, repoClient repo.Client) (*Bot, error) {
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
		bot:  b,
		log:  logger.With("bot"),
		repo: repoClient,
	}

	bot.applyNotifications(cfg)

	return bot, nil
}

type Bot struct {
	bot *telebot.Bot

	ctx context.Context
	log logger.Logger

	txm  txn.Manager
	repo repo.Client

	notifyBefore []int64
	notifyPeriod time.Duration
}

func (b *Bot) Run(ctx context.Context) error {
	b.ctx = ctx

	b.txm = txn.NewManager(
		ctx,
		b.log.With("txn"),
		b.repo,
		time.Second*3,
	)

	b.setupHandlers()
	go b.bot.Start()
	b.runNotifier()
	return nil
}

func (b *Bot) Stop() {
	b.bot.Stop()
}
