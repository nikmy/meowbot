package telegram

import (
	"fmt"
	"strings"
	"time"

	"github.com/vitaliy-ukiru/fsm-telebot"
	"github.com/vitaliy-ukiru/fsm-telebot/storages/memory"
	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/users"
	"github.com/nikmy/meowbot/pkg/errors"
)

const (
	candStr  = "Кандидат"
	interStr = "Собеседующий"
)

const (
	initialState                  = fsm.DefaultState
	chooseRoleState     fsm.State = "chooseRole"
	chooseIntervalState fsm.State = "chooseInterval"
)

var (
	iamCandidateBtn   = telebot.ReplyButton{Text: candStr}
	iamInterviewerBtn = telebot.ReplyButton{Text: interStr}
)

func (b *Bot) setupHandlers() {
	manager := fsm.NewManager(
		b.Bot,
		nil,
		memory.NewStorage(),
		nil,
	)

	manager.Bind("/me", fsm.AnyState, b.GetMe)
	manager.Bind("/start", fsm.AnyState, b.Start)
	manager.Bind("/match", fsm.AnyState, b.Match)
	manager.Bind(telebot.OnText, chooseRoleState, b.ChooseRole)
	manager.Bind(telebot.OnText, chooseIntervalState, b.TryInterval)

	manager.Bind(&iamCandidateBtn, chooseRoleState, b.ChooseRole)
	manager.Bind(&iamInterviewerBtn, chooseRoleState, b.ChooseRole)
}

func (b *Bot) setState(s fsm.Context, target fsm.State) {
	err := s.Set(target)
	if err != nil {
		b.logger.Warn(errors.WrapFail(err, "set state to \"%s\"", target))
	}
}

func (b *Bot) final(c telebot.Context, s fsm.Context, msg string, opts ...any) error {
	b.setState(s, initialState)
	return c.Send(msg, opts)
}

func (b *Bot) GetMe(c telebot.Context, s fsm.Context) error {
	user, err := b.users.Get(b.ctx, c.Sender().Username)
	if err != nil {
		b.logger.Error(errors.WrapFail(err, "do Users.Get request"))
		return b.final(c, s, "Что-то пошло не так")
	}

	if user == nil {
		b.setState(s, chooseRoleState)
		return c.Send("Выберите свою роль", telebot.SendOptions{
			ReplyMarkup: &telebot.ReplyMarkup{
				ForceReply: true,
				ReplyKeyboard: [][]telebot.ReplyButton{
					{iamCandidateBtn, iamInterviewerBtn},
				},
			},
		})
	}

	roleStr := candStr
	if user.Role == users.Interviewer {
		roleStr = interStr
	}

	return b.final(c, s, fmt.Sprintf("Вы — %s", roleStr))
}

func (b *Bot) Start(c telebot.Context, s fsm.Context) error {
	b.setState(s, initialState)
	return c.Send(USAGE)
}

func (b *Bot) Match(c telebot.Context, s fsm.Context) error {
	args := strings.Fields(c.Text())
	if len(args) != 2 {
		return b.final(c, s, "Введите ID собеседования через пробел.")
	}

	id := args[1]
	found, err := b.interviews.Find(b.ctx, id)
	if err != nil {
		b.logger.Error(errors.WrapFail(err, "do Interviews.Find request"))
		b.setState(s, initialState)
		return c.Send("Что-то пошло не так :(\nПопробуйте позже.")
	}

	if !found {
		b.setState(s, initialState)
		return c.Send("Такого ID собеседования нет.")
	}

	user, err := b.users.Get(b.ctx, c.Sender().Username)
	if err != nil {
		b.logger.Error(errors.WrapFail(err, "do Users.Get request"))
		return b.final(c, s, "Что-то пошло не так")
	}

	if user == nil || user.Role != users.Candidate {
		b.setState(s, chooseRoleState)
		return b.final(c, s, "Это функционал только для кандидатов")
	}

	b.setState(s, chooseIntervalState)
	return c.Send(
		"Введите дату и время в формате ДД ММ ГГГГ ЧЧ ММ:",
		telebot.SendOptions{
			ReplyMarkup: &telebot.ReplyMarkup{
				ForceReply: true,
			},
		},
	)
}

func (b *Bot) ChooseRole(c telebot.Context, s fsm.Context) error {
	if c.Text() == candStr {
		err := b.users.Add(b.ctx, &users.User{
			Intervals: nil,
			Username:  c.Sender().Username,
			Role:      users.Candidate,
		})
		if err != nil {
			b.logger.Error(errors.WrapFail(err, "do Users.Add request"))
			return c.Send("Что-то пошло не так")
		}
		return c.Send("Теперь вы кандидат! Используйте /match для подбора времени")
	}

	if c.Text() == interStr {
		err := b.users.Add(b.ctx, &users.User{
			Intervals: nil,
			Username:  c.Sender().Username,
			Role:      users.Interviewer,
		})
		if err != nil {
			b.logger.Error(errors.WrapFail(err, "do Users.Add request"))
			return c.Send("Что-то пошло не так")
		}
		return c.Send("Теперь вы собеседующий! Когда назначат собеседование, я пришлю уведомление")
	}

	return b.final(c, s, "Я не понимаю :(")
}
func (b *Bot) TryInterval(c telebot.Context, s fsm.Context) error {
	left, err := time.Parse("02 01 2006 15 04", c.Text())
	if err != nil {
		return c.Send("Плохой формат даты.")
	}

	interval := [2]int64{left.UnixMilli(), left.Add(time.Hour).UnixMilli()}

	free, err := b.users.Match(b.ctx, interval)
	if err != nil {
		b.logger.Error(errors.WrapFail(err, "do Users.Mathc request"))
		return c.Send("Что-то пошло не так, попробуйте позже.")
	}

	for len(free) > 0 {
		assigned, err := b.users.Assign(b.ctx, free[0], interval)
		if err != nil {
			b.logger.Warn(errors.WrapFail(err, "assign interval to interviewer"))
		}
		if assigned {
			break
		}
		free = free[1:]
	}

	if len(free) == 0 {
		return b.final(
			c, s,
			"На выбранный слот совпадений не нашлось :(\nПопробуйте изменить дату или время.",
			telebot.SendOptions{
				ReplyMarkup: &telebot.ReplyMarkup{
					ForceReply: true,
				},
			},
		)
	}

	return b.final(c, s, "Назначили собеседование на %s", left.Format(time.RFC850))
}
