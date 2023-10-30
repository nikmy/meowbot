package callbacks

import (
	"github.com/nikmy/meowbot/pkg/tools/uid"
	"gopkg.in/telebot.v3"
	"sync"
)

type Handler[State any] func(unique string, state *State) error

type EventMapper[State any] struct {
	mappers map[string]*EventMapper[State]
	handler Handler[State]
	id      string
	mu      sync.RWMutex
}

func New[State any]() *EventMapper[State] {
	return &EventMapper[State]{
		mappers: map[string]*EventMapper[State]{},
	}
}

func (m *EventMapper[State]) SetID(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.id = id
}

func (m *EventMapper[State]) BindInline(btn *telebot.InlineButton) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	btn.Unique = uid.Generate()
	return nil
}

func (m *EventMapper[State]) On(event string) *EventMapper[State] {
	m.mu.Lock()
	defer m.mu.Unlock()

	mapper, ok := m.mappers[event]
	if !ok {
		mapper = New[State]()
		m.mappers[event] = mapper
	}

	return mapper
}

func (m *EventMapper[State]) Attach(event string, mapper *EventMapper[State]) *EventMapper[State] {
	if mapper == nil {
		return m
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mappers[event] = mapper
	return mapper
}

func (m *EventMapper[State]) Do(h Handler[State]) *EventMapper[State] {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handler = h
	return m
}
