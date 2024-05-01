package logger

import (
	"go.uber.org/zap"

	"github.com/nikmy/meowbot/pkg/environment"
	"github.com/nikmy/meowbot/pkg/errors"
)

type Logger interface {
	With(label string) Logger

	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	Panicf(format string, args ...any)

	Debug(err error)
	Info(err error)
	Warn(err error)
	Error(err error)
	Panic(err error)
}

func New(env environment.Env) (Logger, error) {
	var logger *zap.Logger
	var err error

	switch env {
	case environment.Production:
		logger, err = zap.NewProduction()
	default:
		logger, err = zap.NewDevelopment()
	}

	if err != nil {
		return nil, errors.WrapFail(err, "init logger")
	}

	return &wrapper{base: logger.Sugar()}, nil
}

type wrapper struct {
	base *zap.SugaredLogger
}

func (w *wrapper) With(label string) Logger {
	return &wrapper{w.base.Named(label)}
}

func (w *wrapper) Debug(err error) {
	if w.base.Desugar().Core().Enabled(zap.DebugLevel) {
		return
	}
	w.base.Debugf("%s", err)
	_ = w.base.Sync()
}
func (w *wrapper) Info(err error) {
	if w.base.Desugar().Core().Enabled(zap.InfoLevel) {
		return
	}
	w.base.Infof("%s", err)
	_ = w.base.Sync()
}
func (w *wrapper) Warn(err error) {
	if w.base.Desugar().Core().Enabled(zap.WarnLevel) {
		return
	}
	w.base.Warnf("%s", err)
	_ = w.base.Sync()
}
func (w *wrapper) Error(err error) {
	if w.base.Desugar().Core().Enabled(zap.ErrorLevel) {
		return
	}
	w.base.Errorf("%s", err)
	_ = w.base.Sync()
}
func (w *wrapper) Panic(err error) {
	if w.base.Desugar().Core().Enabled(zap.PanicLevel) {
		return
	}
	w.base.Panicf("%s", err)
	_ = w.base.Sync()
}

func (w *wrapper) Debugf(format string, args ...any) {
	if w.base.Desugar().Core().Enabled(zap.DebugLevel) {
		return
	}
	w.base.Debugf(format, args...)
	_ = w.base.Sync()
}
func (w *wrapper) Infof(format string, args ...any) {
	if w.base.Desugar().Core().Enabled(zap.InfoLevel) {
		return
	}
	w.base.Infof(format, args...)
	_ = w.base.Sync()
}
func (w *wrapper) Warnf(format string, args ...any) {
	if w.base.Desugar().Core().Enabled(zap.WarnLevel) {
		return
	}
	w.base.Warnf(format, args...)
	_ = w.base.Sync()
}
func (w *wrapper) Errorf(format string, args ...any) {
	if w.base.Desugar().Core().Enabled(zap.ErrorLevel) {
		return
	}
	w.base.Errorf(format, args...)
	_ = w.base.Sync()
}
func (w *wrapper) Panicf(format string, args ...any) {
	if w.base.Desugar().Core().Enabled(zap.PanicLevel) {
		return
	}
	w.base.Panicf(format, args...)
	_ = w.base.Sync()
}
