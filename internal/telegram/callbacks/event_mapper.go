package callbacks

import "gopkg.in/telebot.v3"

type Handler[State any] func(unique string, state *State) error

type EventMapper[State any] struct {
}

func (m *EventMapper[State]) On(event string) *EventMapper[State] {
	return m
}

func (m *EventMapper[State]) Do(h Handler[State]) *EventMapper[State] {
	return m
}

func (m *EventMapper[State]) BindInline(btn *telebot.InlineButton, s string) error {
	return nil
}
