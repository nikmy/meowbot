package telegram

import (
	"github.com/vitaliy-ukiru/fsm-telebot"
	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/repo"
	"github.com/nikmy/meowbot/internal/repo/models"
	"github.com/nikmy/meowbot/pkg/logger"
)

type telebotContext interface {
	telebot.Context
}

type fsmContext interface {
	fsm.Context
}

type repoClient interface {
	repo.Client
}

type interviewsApi interface {
	models.InterviewsRepo
}

type usersApi interface {
	models.UsersRepo
}

type loggerImpl interface {
	logger.Logger
}

type pubsub interface {
	Pull(channel string) ([][]byte, error)
}
