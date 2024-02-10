package callbacks

import "gopkg.in/telebot.v3"

type Mapper interface {
	Do(clb telebot.HandlerFunc)
	BindInline(button *telebot.InlineButton) error
}

func NewEventMapper() *EventMapper {
	return nil
}

type EventMapper struct {
	callback telebot.HandlerFunc
	mappers  map[string]*EventMapper
}

func (m *EventMapper) On(path string) *EventMapper {
	mapper, ok := m.mappers[path]
	if !ok {
		mapper = NewEventMapper()
		m.mappers[path] = mapper
	}

	return mapper
}

func (m *EventMapper) BindInline(button *telebot.InlineButton) error {
	return nil
}

func (m *EventMapper) Do(h telebot.HandlerFunc) {
	m.callback = h
}
