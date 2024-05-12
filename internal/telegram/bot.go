package telegram

import (
	"context"
	"github.com/nikmy/meowbot/internal/repo"
	"github.com/nikmy/meowbot/pkg/errors"
	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/interviews"
	"github.com/nikmy/meowbot/internal/users"
	"github.com/nikmy/meowbot/pkg/logger"
)

func New(
	logger logger.Logger,
	conf Config,
	dbConf repo.Config,
	interviewsSrc repo.DataSource,
	usersSrc repo.DataSource,
) (*Bot, error) {
	b, err := telebot.NewBot(telebot.Settings{
		Token:   conf.Token,
		Updates: 256,
		Poller: &telebot.LongPoller{
			Timeout: conf.PollInterval,
		},
	})
	return &Bot{
		bot:           b,
		logger:        logger,
		dbConf:        dbConf,
		usersSrc:      usersSrc,
		interviewsSrc: interviewsSrc,
	}, err
}

type Bot struct {
	bot *telebot.Bot
	ctx context.Context

	dbConf repo.Config

	usersSrc repo.DataSource
	users    users.API

	interviewsSrc repo.DataSource
	interviews    interviews.API

	logger logger.Logger
}

func (b *Bot) Run(ctx context.Context) error {
	ivRepo, err := interviews.New(ctx, b.logger, b.dbConf, b.interviewsSrc)
	if err != nil {
		return errors.WrapFail(err, "init interviews repo")
	}
	b.interviews = ivRepo

	uRepo, err := users.New(ctx, b.logger, b.dbConf, b.usersSrc)
	if err != nil {
		return errors.WrapFail(err, "init users repo")
	}
	b.users = uRepo

	b.ctx = ctx
	b.setupHandlers()
	go b.bot.Start()
	return nil
}

func (b *Bot) Stop() {
	b.bot.Stop()
}
