package telegram

import (
	"runtime/debug"

	"github.com/vitaliy-ukiru/fsm-telebot"
	"github.com/vitaliy-ukiru/fsm-telebot/storages/memory"
	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/repo/models"
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

	addZoomReadIIDState  fsm.State = "addZoomReadIId"
	addZoomReadLinkState fsm.State = "addZoomReadLink"
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
		"/delInterviewer — удалить интервьюера\n" +
		"/addZoom — добавить ссылку на встречу"
}

func (b *Bot) setupHandlers() {
	manager := fsm.NewManager(
		b.bot,
		nil,
		memory.NewStorage(),
		nil,
	)

	manager.Bind(telebot.OnText, initialState, b.panicHandler(b.start))
	manager.Bind("/start", fsm.AnyState, b.panicHandler(b.start))

	manager.Bind("/show_interviews", fsm.AnyState, b.panicHandler(b.showInterviews))

	manager.Bind("/match", initialState, b.panicHandler(b.runMatch))
	manager.Bind(telebot.OnText, matchReadIIDState, b.panicHandler(b.matchReadIID))
	manager.Bind(telebot.OnText, matchReadIntervalState, b.panicHandler(b.match))

	manager.Bind("/cancel", initialState, b.panicHandler(b.runCancel))
	manager.Bind(telebot.OnText, cancelReadIIDState, b.panicHandler(b.cancel))

	manager.Bind("/create", initialState, b.panicHandler(b.runCreate))
	manager.Bind(telebot.OnText, createReadInfoState, b.panicHandler(b.createReadInfo))
	manager.Bind(telebot.OnText, createReadCTgState, b.panicHandler(b.create))

	manager.Bind("/delete", initialState, b.panicHandler(b.runDelete))
	manager.Bind(telebot.OnText, deleteReadIIDState, b.panicHandler(b.delete))

	manager.Bind("/addInterviewer", initialState, b.panicHandler(b.runAddInterviewer))
	manager.Bind(telebot.OnText, addIntReadTgState, b.panicHandler(b.addInterviewer))
	manager.Bind("/delInterviewer", initialState, b.panicHandler(b.runDelInterviewer))
	manager.Bind(telebot.OnText, delIntReadTgState, b.panicHandler(b.delInterviewer))

	manager.Bind("/addZoom", initialState, b.panicHandler(b.runAddZoom))
	manager.Bind(telebot.OnText, addZoomReadIIDState, b.panicHandler(b.addZoomReadIID))
	manager.Bind(telebot.OnText, addZoomReadLinkState, b.panicHandler(b.addZoom))
}

func (b *Bot) panicHandler(h fsm.Handler) fsm.Handler {
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
		b.log.Error(errors.WrapFail(err, "set state to \"%s\"", target))
	}
}

func (b *Bot) final(c telebot.Context, s fsm.Context, msg string, opts ...any) error {
	err := s.Finish(true)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "finish state"))
	}

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
