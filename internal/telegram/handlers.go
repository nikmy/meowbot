package telegram

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/vitaliy-ukiru/fsm-telebot"
	"github.com/vitaliy-ukiru/fsm-telebot/storages/memory"
	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/repo/models"
	"github.com/nikmy/meowbot/pkg/errors"
)

const (
	initialState = fsm.DefaultState

	matchReadIIDState      fsm.State = "enterIdForMatch"
	matchReadIntervalState fsm.State = "chooseInterval"

	createReadInfoState fsm.State = "crReadInfo"
	createReadCTgState  fsm.State = "crReadTg"

	deleteReadIIDState fsm.State = "delReadIID"

	cancelReadIIDState fsm.State = "cReadIID"
)

const USAGE = "Доступные команды:\n" +
	"/match — подобрать время для собеседования (функционал для кандидата)\n" +
	"/show_interviews — показать все мои собеседования\n" +
	"/create — создать собеседование\n" +
	"/delete — удалить собеседование\n" +
	"/cancel — отменить запланированное собеседование\n" +
	"/join — присоединиться к команде интервьюеров\n" +
	"/leave — перестать быть интервьюером\n"

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

	manager.Bind("/join", initialState, b.joinTeam)
	manager.Bind("/leave", initialState, b.leaveTeam)
}

func (b *Bot) setState(s fsm.Context, target fsm.State) {
	err := s.Set(target)
	if err != nil {
		b.log.Warn(errors.WrapFail(err, "set state to \"%s\"", target))
	}
}

func (b *Bot) final(c telebot.Context, s fsm.Context, msg string, opts ...any) error {
	b.setState(s, initialState)
	return c.Send(msg, opts...)
}

func (b *Bot) fail(c telebot.Context, s fsm.Context, err error) error {
	b.log.Error(err)
	return b.final(c, s, "Что-то пошло не так")
}

func (b *Bot) start(c telebot.Context, s fsm.Context) error {
	sender := c.Sender()
	if sender == nil {
		return b.fail(c, s, errors.Fail("get sender"))
	}

	err := b.users.Upsert(
		b.ctx,
		sender.Username,
		&sender.ID,
		nil,
	)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "save user telegram id"))
		return b.final(
			c, s,
			"Ошибка. Если вы используете бота в первый раз, "+
				"функционал может быть недоступен. Свяжитесь с поддержкой",
		)
	}

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

	if i.CandidateTg != sender.ID {
		return b.final(c, s, "Вы не являетесь кандидатом в этом собеседовании")
	}

	err = s.Update("iid", iid)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "update state with iid"))
	}

	b.setState(s, matchReadIntervalState)
	return c.Send("Введите дату и время в формате ДД ММ ГГГГ ЧЧ ММ ZZZ:")
}

func (b *Bot) match(c telebot.Context, s fsm.Context) error {
	var iid string
	err := s.Get("iid", &iid)
	if err != nil {
		b.log.Debug(err)
		return b.final(c, s, "Ошибка, попробуйте ещё раз")
	}

	left, err := time.Parse("02 01 2006 15 04 MST", c.Text())
	if err != nil {
		b.log.Debug(err)
		return c.Send("Плохой формат даты.")
	}
	left = left.UTC()

	meeting := models.Meeting{left.UnixMilli(), left.Add(time.Hour).UnixMilli()}

	i, err := b.interviews.Find(b.ctx, iid)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "find interview to match"))
	}

	if i == nil {
		return b.final(c, s, "Такого собеседования нет")
	}

	if i.Interval != nil {
		return b.final(c, s, fmt.Sprintf("Собеседование уже назначено на %s", time.UnixMilli(i.Interval[0])))
	}

	free, err := b.users.Match(b.ctx, meeting)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "do Users.Mathc request"))
	}

	sender := c.Sender()
	if sender == nil {
		return b.fail(c, s, errors.Fail("get sender"))
	}

	assigned := false
	for !assigned && len(free) > 0 {
		// TODO: txn
		assigned, err = b.users.Schedule(b.ctx, free[0].Username, meeting)
		if err != nil {
			b.log.Warn(errors.WrapFail(err, "assign interview to interviewer"))
			continue
		}

		if !assigned {
			free = free[1:]
			continue
		}

		err = b.interviews.Schedule(b.ctx, iid, free[0].Telegram, meeting)
		if err != nil {
			b.log.Warn(errors.WrapFail(err, "schedule interview"))
			assigned = false
			continue
		}
	}

	if len(free) == 0 {
		return b.final(
			c, s,
			"На выбранный слот совпадений не нашлось :(\nПопробуйте изменить дату или время.",
		)
	}

	msg := fmt.Sprintf("Назначили собеседование на %s UTC", left.Format(time.DateTime))

	err = b.notify(free[0].Telegram, msg)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "notify interviewer"))
	}

	return b.final(c, s, msg)
}

func (b *Bot) showInterviews(c telebot.Context, s fsm.Context) error {
	sender := c.Sender()
	if sender == nil {
		return b.fail(c, s, errors.Fail("get sender"))
	}

	assigned, err := b.interviews.FindByUser(b.ctx, sender.ID)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "find by candidate"))
	}

	if len(assigned) == 0 {
		return b.final(c, s, "У вас нет назначенных собеседований")
	}

	slices.SortFunc(assigned, func(a, b models.Interview) int {
		return cmp.Or(
			cmp.Compare(a.Interval[0], b.Interval[0]),
			cmp.Compare(a.Interval[1], b.Interval[1]),
		)
	})

	var sb strings.Builder

	for _, i := range assigned {
		sb.WriteRune('`')
		sb.WriteString(i.ID)
		sb.WriteRune('`')

		sb.WriteString(" ")
		sb.WriteString(i.Vacancy)

		sb.WriteString(" (")
		switch sender.ID {
		case i.InterviewerTg:
			sb.WriteRune('И')
		case i.CandidateTg:
			sb.WriteRune('К')
		}
		sb.WriteString("): ")

		if i.Interval == nil {
			sb.WriteString("не запланировано;\n")
			continue
		}

		sb.WriteString(fmt.Sprintf("%s;\n", time.UnixMilli(i.Interval[0]).Format(time.DateTime)))
	}

	return b.final(c, s, sb.String(), &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
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

	sender := c.Sender()
	if sender == nil {
		return b.fail(c, s, errors.Fail("get sender"))
	}

	err = b.users.Upsert(b.ctx, tg, nil, nil)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "upsert user"))
		return b.fail(c, s, err)
	}

	id, err := b.interviews.Create(b.ctx, vac, sender.ID)
	if err != nil {
		b.log.Error(err)
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
	return c.Send("Введите ID собеседования")
}

func (b *Bot) cancel(c telebot.Context, s fsm.Context) error {
	iid := c.Text()

	sender := c.Sender()
	if sender == nil {
		return b.fail(c, s, errors.Fail("get sender"))
	}

	// TODO: txn
	i, err := b.interviews.Find(b.ctx, iid)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "do Interviews.Find request"))
	}

	if i == nil {
		return b.final(c, s, "Собеседование не найдено")
	}

	if i.Interval == nil {
		return b.final(c, s, "Собеседование не запланировано")
	}

	side := models.RoleInterviewer
	if i.CandidateTg == sender.ID {
		side = models.RoleCandidate
	}

	err = b.users.Free(b.ctx, sender.Username, *i.Interval)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "do Users.Free request"))
	}

	err = b.interviews.Cancel(b.ctx, iid, side)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "do Interviews.Cancel request"))
	}
	//if err != nil {
	//	return b.fail(c, s, errors.WrapFail(err, "perform cancel txn"))
	//}
	//if !ok {
	//	return b.fail(c, s, errors.Error("interview cancellation has been aborted"))
	//}

	return b.final(c, s, "Собеседование отменено")
}

func (b *Bot) joinTeam(c telebot.Context, s fsm.Context) error {
	sender := c.Sender()
	if sender == nil {
		return b.fail(c, s, errors.Fail("get sender"))
	}

	mark := true

	err := b.users.Upsert(b.ctx, sender.Username, &sender.ID, &mark)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "do Users.Upsert"))
	}

	return b.final(c, s, "Теперь вы — интервьюер! Ждите собеседований ;)")
}

func (b *Bot) leaveTeam(c telebot.Context, s fsm.Context) error {
	sender := c.Sender()
	if sender == nil {
		return b.fail(c, s, errors.Fail("get sender"))
	}

	mark := false

	err := b.users.Upsert(b.ctx, sender.Username, &sender.ID, &mark)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "do Users.Upsert"))
	}

	return b.final(c, s, "Вы больше не интервьюер (или им не были)")
}
