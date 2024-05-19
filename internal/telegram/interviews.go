package telegram

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/vitaliy-ukiru/fsm-telebot"
	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/repo/models"
	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/txn"
)

func (b *Bot) runMatch(c telebot.Context, s fsm.Context) error {
	b.setState(s, matchReadIIDState)
	return c.Send("Введите ID собеседования")
}

func (b *Bot) matchReadIID(c telebot.Context, s fsm.Context) error {
	iid := c.Text()

	i, err := b.repo.Interviews().Find(b.ctx, iid)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "find interview by id"))
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
			"На выбранный слот совпадений не нашлось :(\n"+
				"Попробуйте изменить дату или время.",
		)
	}

	msg := fmt.Sprintf("Назначили собеседование `%s` на %s", iid, left.Format("02.01.06 15:04:05"))

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

	for idx, i := range assigned {
		sb.WriteString(strconv.Itoa(idx))
		sb.WriteString(". ")

		sb.WriteRune('`')
		sb.WriteString(i.ID)
		sb.WriteRune('`')

		sb.WriteString(": \"")
		sb.WriteString(i.Vacancy)

		sb.WriteString("\"\t")
		switch sender.ID {
		case i.InterviewerTg:
			sb.WriteString("Интер.")
		case i.CandidateTg:
			sb.WriteString("Канд.")
		}
		sb.WriteString(", ")

		if i.Meet == nil {
			sb.WriteString(" не запланировано;\n")
			continue
		}

		timeInfo := fmt.Sprintf(
			" %s %s;\n",
			time.UnixMilli(i.Meet[0]).Format(time.DateTime),
			b.time.ZoneName(),
		)
		sb.WriteString(timeInfo)
	}

	return b.final(c, s, sb.String(), &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
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
		return b.fail(c, s, errors.WrapFail(err, "find interview by id"))
	}

	if i == nil {
		return b.final(c, s, "Собеседование не найдено")
	}

	if i.Meet == nil {
		return b.final(c, s, "Собеседование не запланировано")
	}

	var side models.Role
	switch sender.ID {
	case i.CandidateTg:
		side = models.RoleCandidate
	case i.InterviewerTg:
		side = models.RoleInterviewer
	default:
		return b.final(c, s, "Вы не являетесь участником собеседования")
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

func (b *Bot) cancelInterview(ctx context.Context, interview *models.Interview, side models.Role) (bool, error) {
	if interview.Meet == nil {
		return false, nil
	}

	err := b.repo.Interviews().Cancel(ctx, interview.ID, side)
	if err != nil {
		return false, errors.WrapFail(err, "do Interviews.Cancel request")
	}

	ok, err := b.cancelMeeting(ctx, interview.InterviewerUN, *interview.Meet)
	if err != nil {
		return false, errors.WrapFail(err, "cancel meeting")
	}
	if !ok {
		return false, nil
	}

	ok, err = b.cancelMeeting(ctx, interview.CandidateUN, *interview.Meet)
	if err != nil {
		return false, errors.WrapFail(err, "cancel meeting")
	}
	if !ok {
		return false, nil
	}

	return true, nil
}

func (b *Bot) cancelMeeting(ctx context.Context, username string, meet models.Meeting) (bool, error) {
	user, err := b.repo.Users().Get(ctx, username)
	if err != nil {
		return false, errors.WrapFail(err, "find user")
	}

	if user == nil {
		return false, nil
	}

	meets, found := user.FindAndDeleteMeeting(meet)
	if !found {
		return false, nil
	}

	updated, err := b.repo.Users().UpdateMeetings(ctx, username, meets)
	if err != nil {
		return false, errors.WrapFail(err, "update meetings")
	}

	return updated, nil
}

func (b *Bot) tryAssign(
	ctx context.Context,
	candidate models.User,
	interviewer models.User,
	iid string,
	meet models.Meeting,
) (bool, bool) {
	if candidate.Username == interviewer.Username {
		return false, true
	}

	tx, err := txn.Start(ctx)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "start txn"))
		return false, true
	}
	defer func() {
		err := tx.Close(ctx)
		if err != nil {
			b.log.Warn(errors.WrapFail(err, "close txn"))
		}
	}()

	scheduled, err := b.scheduleMeeting(ctx, candidate.Username, meet)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "schedule meeting for candidate"))
		return false, true
	}
	if !scheduled {
		return false, false
	}

	scheduled, err = b.scheduleMeeting(ctx, interviewer.Username, meet)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "schedule meet for interviewer"))
		return false, true
	}
	if !scheduled {
		return false, true
	}

	err = b.repo.Interviews().Schedule(ctx, iid, candidate, interviewer, meet)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "schedule interview"))
		return false, true
	}

	err = tx.Commit(ctx)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "commit txn"))
		return false, true
	}

	return true, true
}

func (b *Bot) scheduleMeeting(ctx context.Context, username string, meet models.Meeting) (bool, error) {
	user, err := b.repo.Users().Get(ctx, username)
	if err != nil {
		return false, errors.WrapFail(err, "find user")
	}

	if user == nil {
		return false, nil
	}

	insertIdx, can := user.AddMeeting(meet)
	if !can {
		return false, nil
	}
	meets := slices.Insert(user.Assigned, insertIdx, meet)

	assigned, err := b.repo.Users().UpdateMeetings(ctx, username, meets)
	if err != nil {
		return false, errors.WrapFail(err, "update meetings")
	}

	return assigned, nil
}
