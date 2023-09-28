package state

import (
	"go.uber.org/zap"
	tb "gopkg.in/telebot.v3"
)

type State struct {
	tb.Context
	logger *zap.SugaredLogger
}

func (s *State) Send(what string, opts *tb.SendOptions) {
	if err := s.Context.Send(what, opts); err != nil {
		s.logger.Errorf("cannot send telegram message: %s", err)
	}
}
