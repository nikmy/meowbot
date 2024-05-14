package telegram

import (
	"github.com/nikmy/meowbot/internal/interviews"
	"github.com/nikmy/meowbot/internal/users"
	"github.com/nikmy/meowbot/pkg/logger"
	"github.com/vitaliy-ukiru/fsm-telebot"
	"gopkg.in/telebot.v3"
)

type telebotContext interface {
	telebot.Context
}

type fsmContext interface {
	fsm.Context
}

type interviewsApi interface {
	interviews.API
}

type usersApi interface {
	users.API
}

type loggerImpl interface {
	logger.Logger
}
