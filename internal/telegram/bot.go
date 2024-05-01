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

const USAGE = "/me — получить информацию о своей роли\n" +
	"/match <ID> — подоюрать время для собеседования (доступно для кандидата)"

func New(logger logger.Logger, conf Config, mongoConf repo.MongoConfig) (*Bot, error) {
	b, err := telebot.NewBot(telebot.Settings{
		Token:   conf.Token,
		Updates: 256,
		Poller: &telebot.LongPoller{
			Timeout: conf.PollInterval,
		},
	})
	return &Bot{
		bot:     b,
		logger:  logger,
		repoCfg: mongoConf,
	}, err
}

type Bot struct {
	bot *telebot.Bot
	ctx context.Context

	repoCfg    repo.MongoConfig
	users      users.API
	interviews interviews.API
	logger     logger.Logger
}

func (b *Bot) Run(ctx context.Context) error {
	ivRepo, err := interviews.New(ctx, b.logger, b.repoCfg)
	if err != nil {
		return errors.WrapFail(err, "init interviews repo")
	}
	b.interviews = ivRepo

	b.ctx = ctx
	b.setupHandlers()
	go b.bot.Start()
	return nil
}

func (b *Bot) Stop() {
	b.bot.Stop()
}
