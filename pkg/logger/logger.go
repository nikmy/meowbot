package logger

import (
	"go.uber.org/zap"

	"github.com/nikmy/meowbot/pkg/environment"
	"github.com/nikmy/meowbot/pkg/errors"
)

func New(env environment.Env) (*zap.SugaredLogger, error) {
	var logger *zap.Logger
	var err error

	switch env {
	case environment.Production:
		logger, err = zap.NewProduction(zap.AddCallerSkip(1))
	default:
		logger, err = zap.NewDevelopment(zap.AddCallerSkip(1))
	}

	if err != nil {
		return nil, errors.WrapFail(err, "init logger")
	}

	return logger.Sugar(), nil
}
