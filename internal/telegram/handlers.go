package telegram

import (
	"cmp"
	"fmt"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/vitaliy-ukiru/fsm-telebot"
	"github.com/vitaliy-ukiru/fsm-telebot/storages/memory"
	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/repo/models"
	"github.com/nikmy/meowbot/internal/repo/txn"
	"github.com/nikmy/meowbot/pkg/errors"
)

const (
	initialState = fsm.DefaultState

	matchReadIIDState      fsm.State = "matchReadIId"
	matchReadIntervalState fsm.State = "matchReadInt"

	createReadInfoState fsm.State = "crReadInfo"
	createReadCTgState  fsm.State = "crReadTg"

	deleteReadIIDState fsm.State = "delReadIID"

	cancelReadIIDState fsm.State = "cReadIID"

	addIntReadTgState fsm.State = "addIReadTg"
	delIntReadTgState fsm.State = "delIReadTg"
)

func usage(hr bool) string {
	const common = "" +
		"Доступные команды:\n" +
		"/show_interviews — показать все мои собеседования\n" +
		"/match — подобрать время для собеседования, где я - кандидат\n" +
		"/cancel — отменить запланированное собеседование\n"

	if !hr {
		return common
	}

	return common +
		"/create — создать собеседование\n" +
		"/delete — удалить собеседование\n" +
		"/addInterviewer — добавить интервьюера\n" +
		"/delInterviewer — удалить интервьюера\n"
}

func (b *Bot) setupHandlers() {
	manager := fsm.NewManager(
		b.bot,
		nil,
		memory.NewStorage(),
		nil,
	)

	manager.Bind(telebot.OnText, initialState, b.panicGuard(b.start))
	manager.Bind("/start", fsm.AnyState, b.panicGuard(b.start))

	manager.Bind("/show_interviews", fsm.AnyState, b.panicGuard(b.showInterviews))

	manager.Bind("/match", initialState, b.panicGuard(b.runMatch))
	manager.Bind(telebot.OnText, matchReadIIDState, b.panicGuard(b.matchReadIID))
	manager.Bind(telebot.OnText, matchReadIntervalState, b.panicGuard(b.match))

	manager.Bind("/cancel", initialState, b.panicGuard(b.runCancel))
	manager.Bind(telebot.OnText, cancelReadIIDState, b.panicGuard(b.cancel))

	manager.Bind("/create", initialState, b.panicGuard(b.runCreate))
	manager.Bind(telebot.OnText, createReadInfoState, b.panicGuard(b.createReadInfo))
	manager.Bind(telebot.OnText, createReadCTgState, b.panicGuard(b.create))

	manager.Bind("/delete", initialState, b.panicGuard(b.runDelete))
	manager.Bind(telebot.OnText, deleteReadIIDState, b.panicGuard(b.delete))

	manager.Bind("/addInterviewer", initialState, b.panicGuard(b.runAddInterviewer))
	manager.Bind(telebot.OnText, addIntReadTgState, b.panicGuard(b.addInterviewer))
	manager.Bind("/delInterviewer", initialState, b.panicGuard(b.runDelInterviewer))
	manager.Bind(telebot.OnText, delIntReadTgState, b.panicGuard(b.delInterviewer))
}

func (b *Bot) panicGuard(h fsm.Handler) fsm.Handler {
	return func(c telebot.Context, s fsm.Context) error {
		defer func() {
			if r := recover(); r != nil {
				b.log.Errorf("panic caught: %v\nstacktrace: %s", r, string(debug.Stack()))
			}
		}()
		return h(c, s)
	}
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

	known, err := b.repo.Users().Upsert(b.ctx, sender.Username, &sender.ID, nil, nil)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "upsert user on start"))
		return b.final(
			c, s,
			"Ошибка. Если вы используете бота в первый раз, "+
				"функционал может быть недоступен. Свяжитесь с поддержкой",
		)
	}

	err = b.repo.Interviews().FixTg(b.ctx, sender.Username, sender.ID)
	if err != nil {
		b.log.Warn(errors.WrapFail(err, "fix tg"))
	}

	b.setState(s, initialState)
	return c.Send(usage(known != nil && known.Category == models.HRUser))
}

func (b *Bot) runMatch(c telebot.Context, s fsm.Context) error {
	b.setState(s, matchReadIIDState)
	return c.Send("Введите ID собеседования")
}

func (b *Bot) matchReadIID(c telebot.Context, s fsm.Context) error {
	iid := c.Text()

	i, err := b.repo.Interviews().Find(b.ctx, iid)
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

	if i.CandidateUN != sender.Username {
		return b.final(c, s, "Вы не являетесь кандидатом в этом собеседовании")
	}

	err = s.Update("iid", iid)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "update state with iid"))
	}

	b.setState(s, matchReadIntervalState)
	return c.Send("Введите дату и время в формате ДД ММ ГГГГ ЧЧ ММ")
}

func (b *Bot) match(c telebot.Context, s fsm.Context) error {
	var iid string
	err := s.Get("iid", &iid)
	if err != nil {
		b.log.Debug(err)
		return b.final(c, s, "Ошибка, попробуйте ещё раз")
	}

	t, err := time.Parse("02 01 2006 15 04", c.Text())
	if err != nil {
		b.log.Debug(err)
		return b.final(c, s, "Плохой формат даты.")
	}

	left := t.UTC().Add(b.time.UTCDiff())
	if left.Sub(b.time.Now()) < time.Minute {
		return b.final(c, s, "В это время нельзя провести интервью")
	}

	sender := c.Sender()
	if sender == nil {
		return b.fail(c, s, errors.Fail("get sender"))
	}

	cand, err := b.repo.Users().Get(b.ctx, sender.Username)
	if err != nil {
		return b.fail(c, s, err)
	}
	if cand == nil {
		return b.final(c, s, "Мы не знакомы. Попробуйте /start")
	}

	meet := models.Meeting{left.UnixMilli(), left.Add(time.Hour).UnixMilli()}

	_, free := cand.AddMeeting(meet)
	if !free {
		return b.final(c, s, "В это время вы заняты")
	}

	i, err := b.repo.Interviews().Find(b.ctx, iid)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "find interview to match"))
	}

	if i == nil {
		return b.final(c, s, "Такого собеседования нет")
	}

	if i.Meet != nil {
		return b.final(c, s, fmt.Sprintf("Собеседование уже назначено на %s", time.UnixMilli(i.Meet[0])))
	}

	pool, err := b.repo.Users().Match(b.ctx, meet)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "do Users.Mathc request"))
	}

	ctx, cancel, err := b.txm.NewSessionContext(b.ctx, time.Second*10)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "create session context"))
	}
	defer cancel()

	assigned, candFree := false, true
	for candFree && len(pool) > 0 {
		assigned, candFree = b.tryAssign(ctx, *cand, pool[0], iid, meet)
		if b.ctx.Err() != nil {
			b.log.Error(err)
			break
		}
		if assigned {
			break
		}
		pool = pool[1:]
	}

	if !candFree {
		return b.final(c, s, "В это время вы заняты")
	}

	if len(pool) == 0 {
		return b.final(
			c, s,
			"На выбранный слот совпадений не нашлось :(\nПопробуйте изменить дату или время.",
		)
	}

	msg := fmt.Sprintf("Назначили собеседование `%s` на %s", iid, left.Format("02.01.06 15:04:05 MST"))

	err = b.notify(pool[0].Telegram, msg)
	if err != nil {
		b.log.Warn(errors.WrapFail(err, "notify interviewer"))
	}

	return b.final(c, s, msg, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
}

func (b *Bot) showInterviews(c telebot.Context, s fsm.Context) error {
	sender := c.Sender()
	if sender == nil {
		return b.fail(c, s, errors.Fail("get sender"))
	}

	err := b.repo.Interviews().FixTg(b.ctx, sender.Username, sender.ID)
	if err != nil {
		b.log.Warn(errors.WrapFail(err, "fix tg"))
	}

	assigned, err := b.repo.Interviews().FindByUser(b.ctx, sender.Username)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "find by candidate"))
	}

	if len(assigned) == 0 {
		return b.final(c, s, "У вас нет назначенных собеседований")
	}

	slices.SortFunc(assigned, func(a, b models.Interview) int {
		if a.Meet == nil {
			return -1
		}
		if b.Meet == nil {
			return 1
		}
		return cmp.Or(
			cmp.Compare(a.Meet[0], b.Meet[0]),
			cmp.Compare(a.Meet[1], b.Meet[1]),
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

		if i.Meet == nil {
			sb.WriteString("не запланировано;\n")
			continue
		}

		sb.WriteString(fmt.Sprintf("%s;\n", time.UnixMilli(i.Meet[0]).Format(time.DateTime)))
	}

	return b.final(c, s, sb.String(), &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
}

func (b *Bot) runCreate(c telebot.Context, s fsm.Context) error {
	sender := c.Sender()
	if sender == nil {
		return b.fail(c, s, errors.Fail("get sender"))
	}

	isHR := b.checkHR(sender.Username)
	if !isHR {
		return b.denyNotHR(c, s)
	}

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

func (b *Bot) create(c telebot.Context, s fsm.Context) error {
	var vac string
	err := s.Get("vac", &vac)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "get vacancy from state"))
	}

	tg, msg := b.readTg(c)
	if msg != "" {
		return b.final(c, s, msg)
	}

	sender := c.Sender()
	if sender == nil {
		return b.fail(c, s, errors.Fail("get sender"))
	}

	known, err := b.repo.Users().Upsert(b.ctx, tg, nil, nil, nil)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "upsert user"))
	}

	id, err := b.repo.Interviews().Create(b.ctx, vac, tg)
	if err != nil {
		b.log.Error(err)
		return b.fail(c, s, errors.WrapFail(err, "create interview"))
	}

	if known != nil && known.Telegram != 0 {
		err = b.notify(known.Telegram, fmt.Sprintf(
			"Для вас создано новое собеседование на должность %s, id —`%s`.\nИспользуйте /match, чтобы подобрать удобное время",
			vac, id,
		))
		if err != nil {
			b.log.Warn(errors.WrapFail(err, "notify candidate about new interview"))
		}
	}

	return b.final(
		c, s,
		fmt.Sprintf("Создано собеседование с id `%s`", id),
		&telebot.SendOptions{ParseMode: telebot.ModeMarkdown},
	)
}

func (b *Bot) runDelete(c telebot.Context, s fsm.Context) error {
	sender := c.Sender()
	if sender == nil {
		return b.fail(c, s, errors.Fail("get sender"))
	}

	isHR := b.checkHR(sender.Username)
	if !isHR {
		return b.denyNotHR(c, s)
	}

	b.setState(s, deleteReadIIDState)
	return c.Send("Введите ID собеседования")
}

func (b *Bot) delete(c telebot.Context, s fsm.Context) error {
	iid := c.Text()

	ctx, cancel, err := b.txm.NewSessionContext(b.ctx, time.Second*5)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "init session context"))
	}
	defer cancel()

	found, err := b.repo.Interviews().Delete(b.ctx, iid)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "do Interviews.Find request"))
	}

	if found == nil {
		return b.final(c, s, "Такого собеседования нет")
	}

	tx, err := txn.Start(ctx)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "start txn"))
	}
	defer func() {
		err := tx.Close(ctx)
		if err != nil {
			b.log.Warn(errors.WrapFail(err, "close txn"))
		}
	}()

	cancelled, err := b.cancelInterview(ctx, found, models.RoleHR)
	if err != nil {
		return errors.WrapFail(err, "cancel interview")
	}

	err = tx.Commit(ctx)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "commit txn"))
	}

	if !cancelled {
		return b.final(c, s, "Собеседование удалено")
	}

	msg := fmt.Sprintf("Интервью `%s` на должность \"%s\" удалено", found.ID, found.Vacancy)
	err = b.notify(found.CandidateTg, msg)
	if err != nil {
		b.log.Warn(errors.WrapFail(err, "notify candidate about deletion"))
	}

	if found.InterviewerTg == 0 {
		return b.final(c, s, "Собеседование удалено")
	}

	err = b.notify(found.InterviewerTg, msg)
	if err != nil {
		b.log.Warn(errors.WrapFail(err, "notify candidate about deletion"))
	}

	return b.final(c, s, "Собеседование удалено")
}

func (b *Bot) runCancel(c telebot.Context, s fsm.Context) error {
	b.setState(s, cancelReadIIDState)
	return c.Send("Введите ID собеседования")
}

func (b *Bot) cancel(c telebot.Context, s fsm.Context) error {
	iid := c.Text()

	sender := c.Sender()
	if sender == nil {
		return b.fail(c, s, errors.Fail("get sender"))
	}

	ctx, cancel, err := b.txm.NewSessionContext(b.ctx, time.Second*5)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "create session context"))
	}
	defer cancel()

	tx, err := txn.Start(ctx)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "start txn"))
	}
	defer func() {
		err := tx.Close(ctx)
		if err != nil {
			b.log.Warn(errors.WrapFail(err, "close txn"))
		}
	}()

	i, err := b.repo.Interviews().Find(b.ctx, iid)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "do Interviews.Find request"))
	}

	if i == nil {
		return b.final(c, s, "Собеседование не найдено")
	}

	if i.Meet == nil {
		return b.final(c, s, "Собеседование не запланировано")
	}

	side := models.RoleInterviewer
	if i.CandidateTg == sender.ID {
		side = models.RoleCandidate
	}

	scheduled, err := b.cancelInterview(ctx, i, side)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "cancel interview"))
	}
	if !scheduled {
		return b.final(c, s, "Интервью не запланировано")
	}

	switch side {
	case models.RoleInterviewer:
		err = b.notify(i.CandidateTg, fmt.Sprintf("Интервьюер отменил собеседование `%s`", i.ID))
		err = errors.WrapFail(err, "notify candidate about cancel")
	case models.RoleCandidate:
		err = b.notify(i.InterviewerTg, fmt.Sprintf("Кандидат отменил собеседование `%s`", i.ID))
		err = errors.WrapFail(err, "notify candidate about cancel")
	}

	if err != nil {
		return b.fail(c, s, err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "commit txn"))
	}

	return b.final(c, s, "Собеседование отменено")
}

func (b *Bot) runAddInterviewer(c telebot.Context, s fsm.Context) error {
	sender := c.Sender()
	if sender == nil {
		return b.fail(c, s, errors.Fail("get sender"))
	}

	isHR := b.checkHR(sender.Username)
	if !isHR {
		return b.denyNotHR(c, s)
	}

	b.setState(s, addIntReadTgState)
	return c.Send("Введите telegram будущего интервьюера")
}

func (b *Bot) addInterviewer(c telebot.Context, s fsm.Context) error {
	tg, msg := b.readTg(c)
	if msg != "" {
		return b.final(c, s, msg)
	}

	grade := 1

	old, err := b.repo.Users().Upsert(b.ctx, tg, nil, nil, &grade)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "do Users.Upsert"))
	}

	if old.IntGrade > models.GradeNotInterviewer {
		return b.final(c, s, fmt.Sprintf("@%s уже интервьюер", old.Username))
	}

	return b.final(c, s, fmt.Sprintf("Теперь @%s — интервьюер", old.Username))
}

func (b *Bot) runDelInterviewer(c telebot.Context, s fsm.Context) error {
	sender := c.Sender()
	if sender == nil {
		return b.fail(c, s, errors.Fail("get sender"))
	}

	isHR := b.checkHR(sender.Username)
	if !isHR {
		return b.denyNotHR(c, s)
	}

	b.setState(s, delIntReadTgState)
	return c.Send("Введите telegram интервьюера")
}

func (b *Bot) delInterviewer(c telebot.Context, s fsm.Context) error {
	tg, msg := b.readTg(c)
	if msg != "" {
		return b.final(c, s, msg)
	}

	gradeDown := 0

	ctx, cancel, err := b.txm.NewSessionContext(b.ctx, time.Second*5)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "init session context"))
	}
	defer cancel()

	tx, err := txn.Start(ctx)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "start txn"))
	}
	defer func() {
		err := tx.Close(ctx)
		if err != nil {
			b.log.Warn(errors.WrapFail(err, "close txn"))
		}
	}()

	old, err := b.repo.Users().Update(b.ctx, tg, nil, nil, &gradeDown)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "do Users.Upsert"))
	}
	if old == nil {
		return b.final(c, s, "Такого пользователя не существует")
	}

	if old.IntGrade == models.GradeNotInterviewer {
		return b.final(c, s, fmt.Sprintf("@%s уже не интервьюер", old.Username))
	}

	assigned, err := b.repo.Interviews().FindByUser(ctx, tg)
	if err != nil {
		return errors.WrapFail(err, "find interviews assigned to deleted interviewer")
	}

	for i := range assigned {
		if assigned[i].InterviewerUN != tg {
			continue
		}

		_, err := b.cancelInterview(ctx, &assigned[i], models.RoleInterviewer)
		if err != nil {
			return errors.WrapFail(err, "cancel interview assigned to deleted interviewer")
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "commit txn"))
	}

	return b.final(c, s, fmt.Sprintf("@%s больше не интервьюер", old.Username))
}
