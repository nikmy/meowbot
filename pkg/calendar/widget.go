package calendar

import (
	"time"

	tb "gopkg.in/telebot.v3"
)

type Widget struct {
	inlineBinder
	ignore   inlineBinder
	currDate time.Time
	keyboard [][]tb.InlineButton
	language string
	widgetID string
}

func (w *Widget) Keyboard() [][]tb.InlineButton {
	return w.keyboard
}

type setter func(w *Widget)

func AsBinder(b inlineBinder) setter {
	return func(w *Widget) {
		w.inlineBinder = b
	}
}

func AsIgnore(b inlineBinder) setter {
	return func(w *Widget) {
		w.ignore = b
	}
}

func AsLanguage(lang string) setter {
	return func(w *Widget) {
		w.language = lang
	}
}

func AsCurrentDate(today time.Time) setter {
	return func(w *Widget) {
		w.currDate = today
	}
}

func AsID(id string) setter {
	return func(w *Widget) {
		w.widgetID = id
	}
}

func (w *Widget) addButton(btn tb.InlineButton) error {
	err := w.BindInline(&btn)
	if err != nil {
		return err
	}
	w.keyboard = append(w.keyboard, []tb.InlineButton{btn})
	return nil
}

func (w *Widget) getWeekdaysDisplayNames() [7]string {
	if w.language == "ru" {
		return ruWeekdays
	}
	return enWeekdays
}

func (w *Widget) getMonthDisplayName(m time.Month) string {
	if w.language == "ru" {
		return ruMonths[m]
	}
	return m.String()
}
