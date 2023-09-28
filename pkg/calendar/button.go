package calendar

import (
	"fmt"
	"strconv"
	"time"

	tb "gopkg.in/telebot.v3"
)

func ReturnButton(w *Widget) error {
	return w.addButton(tb.InlineButton{
		Text: "Вернуться назад",
		Data: fmt.Sprintf("%s/back", w.widgetID),
	})
}

func YearButton(w *Widget) error {
	year := strconv.Itoa(w.currDate.Year())
	return w.addButton(tb.InlineButton{
		Text: year,
		Data: fmt.Sprintf("%s/sY/%s", w.widgetID, w.currDate.Format(dateFormat)),
	})
}

func YearMonthButton(w *Widget) error {
	year, month := strconv.Itoa(w.currDate.Year()), w.currDate.Month()

	return w.addButton(tb.InlineButton{
		Text: fmt.Sprintf("%s %s", w.getMonthDisplayName(month), year),
		Data: fmt.Sprintf("%s/sM/%s", w.widgetID, w.currDate.Format(dateFormat)),
	})
}

func DateButton(w *Widget) error {
	y, m, d := w.currDate.Date()
	return w.addButton(tb.InlineButton{
		Text: fmt.Sprintf("%02d.%02d.%04d", d, m, y),
		Data: fmt.Sprintf("%s/sD/%s", w.widgetID, beginningOfMonth(w.currDate).Format(dateFormat)),
	})
}

func IntervalButton(w *Widget) error {
	y, m, d := w.currDate.Date()
	currInt := beginningOf(bigInterval, w.currDate)
	hour, lastHour := currInt.Hour(), nextStep(bigInterval-time.Hour, currInt).Hour()
	return w.addButton(tb.InlineButton{
		Text: fmt.Sprintf("%02d.%02d.%04d %d:00 - %d:59", d, m, y, hour, lastHour),
		Data: fmt.Sprintf("%s/sI/%s", w.widgetID, beginningOfDay(w.currDate).Format(dateFormat)),
	})
}

func HourButton(w *Widget) error {
	y, m, d := w.currDate.Date()
	h := w.currDate.Hour()
	return w.addButton(tb.InlineButton{
		Text: fmt.Sprintf("%02d.%02d.%04d %02d:00 - %02d:59", d, m, y, h, h),
		Data: fmt.Sprintf("%s/sH/%s", w.widgetID, beginningOf(bigInterval, w.currDate).Format(dateFormat)),
	})
}
