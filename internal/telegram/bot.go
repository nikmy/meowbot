package telegram

import (
	"cmp"
	"context"
	"slices"
	"time"

	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/interviews"
	"github.com/nikmy/meowbot/internal/users"
	"github.com/nikmy/meowbot/pkg/logger"
)

func New(
	logger logger.Logger,
	conf Config,
	interviews interviews.API,
	users users.API,
) (*Bot, error) {
	b, err := telebot.NewBot(telebot.Settings{
		Token:   conf.Token,
		Updates: 256,
		Poller: &telebot.LongPoller{
			Timeout: conf.PollInterval,
		},
	})
	if err != nil {
		return nil, err
	}

	notifyBefore := make([]int64, len(conf.NotifyBefore))
	for i := range conf.NotifyBefore {
		notifyBefore[i] = conf.NotifyBefore[i].Milliseconds()
	}

	slices.SortFunc(notifyBefore, cmp.Compare[int64])

	return &Bot{
		bot:        b,
		log:        logger,
		users:      users,
		interviews: interviews,

		notifyBefore: notifyBefore,
		notifyPeriod: conf.NotifyPeriod,
	}, err
}

type Bot struct {
	bot *telebot.Bot

	ctx context.Context
	log logger.Logger

	users      users.API
	interviews interviews.API

	notifyBefore []int64
	notifyPeriod time.Duration
}

func (b *Bot) Run(ctx context.Context) error {
	b.ctx = ctx
	b.setupHandlers()
	go b.bot.Start()

	if b.notifyPeriod > 0 && len(b.notifyBefore) > 0 {
		go b.watch()
	}

	return nil
}

func (b *Bot) Stop() {
	b.bot.Stop()
}
