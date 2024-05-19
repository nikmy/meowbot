package telegram

import (
	"fmt"
	"time"

	"github.com/vitaliy-ukiru/fsm-telebot"
	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/repo/models"
	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/txn"
)

func (b *Bot) readTg(c telebot.Context) (string, string) {
	tg := c.Text()
	if len(tg) < 2 || tg[0] != '@' {
		return "", "Некорректный telegram"
	}
	return tg[1:], ""
}

func (b *Bot) denyNotHR(c telebot.Context, s fsm.Context) error {
	return b.final(c, s, "Это может сделать только HR сотрудник")
}

func (b *Bot) checkHR(username string) bool {
	user, err := b.repo.Users().Get(b.ctx, username)
	if err != nil {
		b.log.Warn(errors.WrapFail(err, "get user for checking HR permissions"))
		return false
	}

	return user.Category >= models.HRUser
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
		return b.fail(c, s, errors.WrapFail(err, "find interview by id"))
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

func (b *Bot) runAddZoom(c telebot.Context, s fsm.Context) error {
	sender := c.Sender()
	if sender == nil {
		return b.fail(c, s, errors.Fail("get sender"))
	}

	isHR := b.checkHR(sender.Username)
	if !isHR {
		return b.denyNotHR(c, s)
	}

	b.setState(s, addZoomReadIIDState)
	return c.Send("Введите id собеседования")
}

func (b *Bot) addZoomReadIID(c telebot.Context, s fsm.Context) error {
	iid := c.Text()
	found, err := b.repo.Interviews().Find(b.ctx, iid)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "find interview by id"))
	}

	if found == nil {
		return b.final(c, s, "Такого собеседования нет")
	}

	err = s.Update("iid", iid)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "update state with iid"))
	}

	b.setState(s, addZoomReadLinkState)
	return c.Send("Введите ссылку на встречу")
}

func (b *Bot) addZoom(c telebot.Context, s fsm.Context) error {
	var iid string
	err := s.Get("iid", &iid)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "get iid from state"))
	}

	link := c.Text()

	err = b.repo.Interviews().Update(b.ctx, iid, nil, nil, nil, &link)
	if err != nil {
		return b.fail(c, s, errors.WrapFail(err, "update interview"))
	}

	return b.final(c, s, "Ссылка добавлена")
}
