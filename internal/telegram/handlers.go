package telegram

import (
	"fmt"
	"github.com/nikmy/meowbot/internal/interviews"
	"strings"
	"text/template"
	"time"

	"github.com/vitaliy-ukiru/fsm-telebot"
	"github.com/vitaliy-ukiru/fsm-telebot/storages/memory"
	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/users"
	"github.com/nikmy/meowbot/pkg/errors"
)

const (
	initialState = fsm.DefaultState

	matchReadIIDState      fsm.State = "enterIdForMatch"
	matchReadIntervalState fsm.State = "chooseInterval"

	createReadInfoState fsm.State = "crReadInfo"
	createReadCTgState fsm.State = "crReadTg"

	deleteReadIIDState fsm.State = "delReadIID"

	cancelReadIIDState fsm.State = "cReadIID"
)

const USAGE = "Доступные команды:\n" +
	"/match — подобрать время для собеседования (функционал для кандидата)\n" +
	"/show_interviews — показать все мои собеседования\n" +
	"/create — создать собеседование\n" +
	"/delete — удалить собеседование\n" +
	"/cancel — отменить запланированное собеседование"

func (b *Bot) setupHandlers() {
	manager := fsm.NewManager(
		b.bot,
		nil,
		memory.NewStorage(),
		nil,
	)

	manager.Bind(telebot.OnText, initialState, b.start)
	manager.Bind("/start", fsm.AnyState, b.start)

	manager.Bind("/show_interviews", fsm.AnyState, b.showInterviews)

	manager.Bind("/match", initialState, b.startMatch)
	manager.Bind(telebot.OnText, matchReadIIDState, b.matchReadIID)
	manager.Bind(telebot.OnText, matchReadIntervalState, b.match)

	manager.Bind("/create", initialState, b.startCreate)
	manager.Bind(telebot.OnText, createReadInfoState, b.createReadInfo)
	manager.Bind(telebot.OnText, createReadCTgState, b.createReadCandidate)

	manager.Bind("/delete", initialState, b.startDelete)
	manager.Bind(telebot.OnText, deleteReadIIDState, b.delete)

	manager.Bind("/cancel", initialState, b.startCancel)
	manager.Bind(telebot.OnText, cancelReadIIDState, b.cancel)
}

func (b *Bot) setState(s fsm.Context, target fsm.State) {
	err := s.Set(target)
	if err != nil {
		b.logger.Warn(errors.WrapFail(err, "set state to \"%s\"", target))
	}
}

func (b *Bot) final(c telebot.Context, s fsm.Context, msg string, opts ...any) error {
	b.setState(s, initialState)
	return c.Send(msg, opts...)
}

func (b *Bot) fail(c telebot.Context, s fsm.Context, err error) error {
	b.logger.Error(err)
	return b.final(c, s, "Что-то пошло не так")
}

func (b *Bot) start(c telebot.Context, s fsm.Context) error {
	b.setState(s, initialState)
	return c.Send(USAGE)
}

func (b *Bot) startMatch(c telebot.Context, s fsm.Context) error {
	b.setState(s, matchReadIIDState)
	return c.Send("Введите ID собеседования")
}

func (b *Bot) matchReadIID(c telebot.Context, s fsm.Context) error {
	iid := c.Text()

	i, err := b.interviews.Find(b.ctx, iid)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "do Interviews.Find request"))
	}

	if i == nil {
		return b.final(c, s, "Такого собеседования нет")
	}

	sender := c.Sender()
	if sender == nil {
		return b.fail(c, s, errors.Fail("get sender"))
	}

	if i.CandidateTg != sender.Username {
		return b.final(c, s, "Вы не являетесь кандидатом в этом собеседовании")
	}

	err = s.Update("iid", iid)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "update state with iid"))
	}

	b.setState(s, matchReadIntervalState)
	return c.Send("Введите дату и время в формате ДД ММ ГГГГ ЧЧ ММ:")
}

func (b *Bot) match(c telebot.Context, s fsm.Context) error {
	var iid string
	err := s.Get("iid", &iid)
	if err != nil {
		b.logger.Debug(err)
		return b.final(c, s, "Ошибка, попробуйте ещё раз")
	}

	left, err := time.Parse("02 01 2006 15 04", c.Text())
	if err != nil {
		b.logger.Debug(err)
		return c.Send("Плохой формат даты.")
	}

	interval := [2]int64{left.UnixMilli(), left.Add(time.Hour).UnixMilli()}

	free, err := b.users.Match(b.ctx, interval)
	if err != nil {
		b.logger.Error(errors.WrapFail(err, "do Users.Mathc request"))
		return b.final(c, s, "Что-то пошло не так, попробуйте позже.")
	}

	interview := users.Interview{
		ID:       iid,
		TimeSlot: interval,
	}

	sender := c.Sender()
	if sender == nil {
		return b.fail(c, s, errors.Fail("get sender"))
	}

	for len(free) > 0 {
		assigned, err := b.users.Assign(b.ctx, sender.Username, free[0].Username, interview, func() error {
			return b.interviews.Schedule(b.ctx, iid, free[0].Username, interval)
		})
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
		)
	}

	return b.final(c, s, "Назначили собеседование на %s", left.Format(time.RFC850))
}

var interviewListTmpl = template.Must(
	template.New("interviewList").Parse(
		`Собеседования, где я кандидат:{{ range index . 1 }}
{{ . }}{{ end }}
Собеседования, где я интервьюер:{{ range index . 0 }}
 {{ . }}{{ end }}`),
)

func (b *Bot) showInterviews(c telebot.Context, s fsm.Context) error {
	sender := c.Sender()
	if sender == nil {
		return b.fail(c, s, errors.Fail("get sender"))
	}

	user, err := b.users.Get(b.ctx, sender.Username)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "get user"))
	}

	if user == nil {
		return b.final(c, s, "У вас нет назначенных собеседований")
	}

	asCandidate, err := b.interviews.FindByCandidate(b.ctx, sender.Username)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "find by candidate"))
	}

	if len(asCandidate) == 0 && len(user.Assigned) == 0 {
		return b.final(c, s, "У вас нет назначенных собеседований")
	}

	var formatted [2][]string
	for _, i := range asCandidate {
		if i.Interval[0] == 0 {
			formatted[interviews.RoleCandidate] = append(formatted[interviews.RoleCandidate], "не назначено")
		}
		formatted[interviews.RoleCandidate] = append(formatted[interviews.RoleCandidate], fmt.Sprintf(
			"%s — %s",
			time.UnixMilli(i.Interval[0]).Format(time.DateTime),
			time.UnixMilli(i.Interval[1]).Format(time.DateTime),
		))
	}

	for _, i := range user.Assigned {
		formatted[interviews.RoleInterviewer] = append(formatted[i.Role], fmt.Sprintf(
			"%s — %s",
			time.UnixMilli(i.TimeSlot[0]).Format(time.DateTime),
			time.UnixMilli(i.TimeSlot[1]).Format(time.DateTime),
		))
	}

	var sb strings.Builder
	err = interviewListTmpl.Execute(&sb, formatted)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "make interview list response"))
	}

	return b.final(c, s, sb.String())
}

func (b *Bot) startCreate(c telebot.Context, s fsm.Context) error {
	b.setState(s, createReadInfoState)
	return c.Send("Введите название вакансии")
}

func (b *Bot) createReadInfo(c telebot.Context, s fsm.Context) error {
	vac := c.Text()

	err := s.Update("vac", vac)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "update state with vac"))
	}

	b.setState(s, createReadCTgState)
	return c.Send("Введите telegram кандидата")
}

func (b *Bot) createReadCandidate(c telebot.Context, s fsm.Context) error {
	var vac string
	err := s.Get("vac", &vac)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "get vacancy from state"))
	}

	tg := c.Text()
	if len(tg) < 2 || tg[0] != '@' {
		return b.final(c, s, "Некорректный telegram")
	}
	tg = tg[1:]

	u, err := b.users.Get(b.ctx, tg)
	if err == nil && u == nil {
		b.logger.Debugf("creating user '%s'", tg)
		err = b.users.Add(b.ctx, &users.User{Username: tg})
	}

	if err != nil {
		b.logger.Error(err)
		return b.fail(c, s, err)
	}

	id, err := b.interviews.Create(b.ctx, vac, tg)
	if err != nil {
		b.logger.Error(err)
		return b.fail(c, s, errors.WrapFail(err, "create interview"))
	}

	return b.final(
		c, s,
		fmt.Sprintf("Создано собеседование с id `%s`", id),
		&telebot.SendOptions{ParseMode: telebot.ModeMarkdown},
	)
}

func (b *Bot) startDelete(c telebot.Context, s fsm.Context) error {
	b.setState(s, deleteReadIIDState)
	return c.Send("Введите ID собеседования")
}

func (b *Bot) delete(c telebot.Context, s fsm.Context) error {
	iid := c.Text()

	found, err := b.interviews.Delete(b.ctx, iid)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "do Interviews.Find request"))
	}

	if !found {
		return b.final(c, s, "Такого собеседования нет")
	}

	return b.final(c, s, "Собеседование удалено")
}

func (b *Bot) startCancel(c telebot.Context, s fsm.Context) error {
	b.setState(s, cancelReadIIDState)
	return c.Send("")
}

func (b *Bot) cancel(c telebot.Context, s fsm.Context) error {
	iid := c.Text()

	i, err := b.interviews.Find(b.ctx, iid)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "do Interviews.Find request"))
	}

	if i == nil {
		return b.final(c, s, "Собеседование не найдено")
	}

	sender := c.Sender()
	if sender == nil {
		return b.fail(c, s, errors.Fail("get sender"))
	}

	side := interviews.RoleInterviewer
	if i.CandidateTg[1:] == sender.Username {
		side = interviews.RoleCandidate
	}

	err = b.interviews.Cancel(b.ctx, iid, side)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "cancel interview"))
	}

	// TODO: notify other side
	return b.final(c, s, "Собеседование отменено")
}
