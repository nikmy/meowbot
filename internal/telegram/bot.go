package telegram

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/repo"
	"github.com/nikmy/meowbot/pkg/txn"
)

func New(log *zap.SugaredLogger, cfg Config, repoClient repo.Client) (*Bot, error) {
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
		log:  log.Named("bot"),
		repo: repoClient,
		time: stdTime{
			zoneName: cfg.TimeZoneConfig.Name,
			utcDiff:  cfg.UTCDiff,
		},
		txm: txn.NewManager(repoClient),
	}

	bot.applyNotifications(cfg)

	return bot, nil
}

type Bot struct {
	bot *telebot.Bot

	ctx context.Context
	log *zap.SugaredLogger

	txm  txn.Manager
	repo repo.Client

	notifyBefore []int64
	notifyPeriod time.Duration

	time timeProvider
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
