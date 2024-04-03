package calendar

import (
	"fmt"
	"strings"
	"time"

	tb "gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/telegram/callbacks"
	"github.com/nikmy/meowbot/pkg/builder"
	"github.com/nikmy/meowbot/pkg/logger"
	"github.com/pkg/errors"
)

func SetupHandler(
	log logger.Logger,
	ignoreChan string,
	returner func(tb.Context, string, time.Time),
) {
	h := &handler{
		callbacks:  callbacks.NewEventMapper(),
		logger:     log.With("calendar"),
		ignoreChan: ignoreChan,
		returner:   returner,
	}

	h.callbacks.On("choose_datetime").Do(h.chooseDatetime)
}

type handler struct {
	callbacks  *callbacks.EventMapper
	logger     logger.Logger
	ignoreChan string
	returner   func(tb.Context, string, time.Time)
}

func (h *handler) newWidgetBuilder(wid string, state tb.Context, currDate time.Time) *builder.Builder[Widget] {
	return builder.New[Widget]().
		Use(AsCurrentDate(currDate)).
		Use(AsLanguage(state.Sender().LanguageCode)).
		//Use(AsBinder(h.callbacks.On("choose_datetime"))).
		//Use(AsIgnore(h.callbacks.On(h.ignoreChan))).
		Use(AsID(wid))
}

func (h *handler) chooseDatetime(state tb.Context) error {
	args := strings.Split(state.Callback().Data, "|")
	if len(args) != 2 {
		return errors.New("calendar: wrong callback data")
	}

	args = strings.Split(args[1], "/")
	if len(args) < 2 {
		return errors.New("calendar: wrong callback data")
	}

	wid, cmd := args[0], args[1]

	if cmd == "back" {
		// выходим из календаря, не выбрав время
		h.returner(state, wid, time.Time{})
		return nil
	}

	if len(args) != 3 {
		return errors.New("calendar: wrong callback data")
	}

	currDate, err := time.Parse(dateFormat, args[2])
	if err != nil {
		return errors.New("calendar: wrong callback data: failed to parse time")
	}

	switch cmd {
	case "sY":
		h.showYears(wid, state)
	case "sM":
		h.showMonths(wid, state, currDate)
	case "sD":
		h.showDays(wid, state, currDate)
	case "sI":
		h.showIntervals(wid, state, currDate)
	case "sH":
		h.showHours(wid, state, currDate)
	case "sS":
		h.showSlots(wid, state, currDate)
	case "cS":
		h.chooseSlot(wid, state, currDate)
	default:
		return fmt.Errorf("calendar: wrong callback data: wrong command %s", cmd)
	}

	return nil
}

func (h *handler) showYears(wid string, state tb.Context) {
	widget, err := h.newWidgetBuilder(wid, state, time.Now()).
		MaybeUse(ChooseYearLayout).
		Get()

	if err != nil {
		h.logger.Warnf("failed on binding inline button: %s", err)
		return
	}

	opts := &tb.SendOptions{
		ReplyMarkup: &tb.ReplyMarkup{
			InlineKeyboard: widget.Keyboard(),
		},
	}

	state.Edit("Выберите год:", opts)
}

func (h *handler) showMonths(wid string, state tb.Context, currDate time.Time) {
	widget, err := h.newWidgetBuilder(wid, state, currDate).
		MaybeUse(YearButton).
		MaybeUse(ChooseMonthLayout).
		Get()

	if err != nil {
		h.logger.Warnf("failed on binding inline button: %s", err)
		return
	}

	opts := &tb.SendOptions{
		ReplyMarkup: &tb.ReplyMarkup{
			InlineKeyboard: widget.Keyboard(),
		},
	}

	state.Edit("Выберите месяц:", opts)
}

func (h *handler) showDays(wid string, state tb.Context, currDate time.Time) {
	widget, err := NewCalendarWidget(
		wid,
		h.callbacks.On("choose_datetime"),
		h.callbacks.On(h.ignoreChan),
		state.Sender().LanguageCode,
		currDate,
	)

	if err != nil {
		h.logger.Warnf("failed on binding inline button: %s", err)
		return
	}

	opts := &tb.SendOptions{
		ReplyMarkup: &tb.ReplyMarkup{
			InlineKeyboard: widget.Keyboard(),
		},
	}

	h.logger.Infof(currDate.String())

	state.Edit("Выберите дату:", opts)
}

func (h *handler) showIntervals(wid string, state tb.Context, day time.Time) {
	widget, err := h.newWidgetBuilder(wid, state, day).
		MaybeUse(DateButton).
		MaybeUse(ChooseIntervalLayout).
		Get()

	if err != nil {
		h.logger.Warnf("failed on binding inline button: %s", err)
		return
	}

	opts := &tb.SendOptions{
		ReplyMarkup: &tb.ReplyMarkup{
			InlineKeyboard: widget.Keyboard(),
		},
	}

	state.Edit("Выберите интервал (ч):", opts)
}

func (h *handler) showHours(wid string, state tb.Context, interval time.Time) {
	widget, err := h.newWidgetBuilder(wid, state, interval).
		MaybeUse(IntervalButton).
		MaybeUse(ChooseHourLayout).
		Get()

	if err != nil {
		h.logger.Warnf("failed on binding inline button: %s", err)
		return
	}

	opts := &tb.SendOptions{
		ReplyMarkup: &tb.ReplyMarkup{
			InlineKeyboard: widget.Keyboard(),
		},
	}

	state.Edit("Выберите час:", opts)
}

func (h *handler) showSlots(wid string, state tb.Context, hour time.Time) {
	widget, err := h.newWidgetBuilder(wid, state, hour).
		MaybeUse(HourButton).
		MaybeUse(ChooseSlotLayout).
		Get()

	if err != nil {
		h.logger.Warnf("failed on binding inline button: %s", err)
		return
	}

	opts := &tb.SendOptions{
		ReplyMarkup: &tb.ReplyMarkup{
			InlineKeyboard: widget.Keyboard(),
		},
	}

	state.Edit("Выберите слот:", opts)
}

func (h *handler) chooseSlot(wid string, state tb.Context, slot time.Time) {
	h.returner(state, wid, slot)
}
