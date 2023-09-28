package calendar

import (
	"fmt"
	"strconv"
	"time"

	tb "gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/pkg/uid"
)

func WeekdaysLayout(w *Widget) error {
	var row []tb.InlineButton

	for _, wd := range w.getWeekdaysDisplayNames() {
		btn := tb.InlineButton{Text: wd}
		if err := w.ignore.BindInline(&btn, uid.Generate()); err != nil {
			return err
		}
		row = append(row, btn)
	}

	w.keyboard = append(w.keyboard, row)
	return nil
}

func ChooseYearLayout(w *Widget) error {
	currYear := beginningOfYear(ceilNow())
	for i := 0; i < 5; i++ {
		year := strconv.Itoa(w.currDate.Year() + i)

		err := w.addButton(tb.InlineButton{
			Text: year,
			Data: fmt.Sprintf("%s/sM/%s", w.widgetID, currYear.AddDate(i, 0, 0).Format(dateFormat)),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func ChooseMonthLayout(w *Widget) error {
	currYear, month := w.currDate.Year(), beginningOfMonth(w.currDate)
	if nowMonth := beginningOfMonth(ceilNow()); month.Before(nowMonth) {
		month = nowMonth
	}

	for month.Year() == currYear {
		err := w.addButton(tb.InlineButton{
			Text: w.getMonthDisplayName(month.Month()),
			Data: fmt.Sprintf("%s/sD/%s", w.widgetID, month.Format(dateFormat)),
		})
		if err != nil {
			return err
		}
		month = month.AddDate(0, 1, 0)
	}

	return nil
}

func ChooseDayLayout(w *Widget) error {
	bOfMonth := beginningOfMonth(w.currDate)
	bOfToday := beginningOfDay(ceilNow())
	daysInMonth := bOfMonth.AddDate(0, 1, -1).Day()

	weekdayNumber := int(bOfMonth.Weekday()+6) % 7

	var row []tb.InlineButton
	if weekdayNumber > 0 {
		row = make([]tb.InlineButton, 0, 7)
		for i := 0; i < weekdayNumber; i++ {
			cell := tb.InlineButton{Text: " "}
			if err := w.ignore.BindInline(&cell, uid.Generate()); err != nil {
				return err
			}
			row = append(row, cell)
		}
	}

	for i := 0; i < daysInMonth; i++ {
		day := bOfMonth.AddDate(0, 0, i)

		if len(row)%7 == 0 {
			w.keyboard = append(w.keyboard, row)
			row = make([]tb.InlineButton, 0, 7)
		}

		if day.Before(bOfToday) {
			cell := tb.InlineButton{Text: " "}
			if err := w.ignore.BindInline(&cell, uid.Generate()); err != nil {
				return err
			}
			row = append(row, cell)
			continue
		}

		dayText := day.Format(dateFormat)

		cell := tb.InlineButton{
			Text: strconv.Itoa(i + 1),
			Data: fmt.Sprintf("%s/sI/%s", w.widgetID, dayText),
		}

		err := w.BindInline(&cell, uid.Generate())
		if err != nil {
			return err
		}

		row = append(row, cell)
	}

	if len(row) > 0 {
		for i := len(row); i < 7; i++ {
			cell := tb.InlineButton{Text: " "}
			if err := w.ignore.BindInline(&cell, uid.Generate()); err != nil {
				return err
			}
			row = append(row, cell)
		}
		w.keyboard = append(w.keyboard, row)
	}

	return nil
}

func ChooseIntervalLayout(w *Widget) error {
	nowInterval := beginningOf(bigInterval, ceilNow())
	begin := beginningOfDay(w.currDate)
	end := beginningOfTomorrow(w.currDate)
	if begin.Before(nowInterval) {
		begin = nowInterval
	}

	var err error
	timeRange(begin, end, bigInterval, func(t time.Time) bool {
		err = w.addButton(tb.InlineButton{
			Text: fmt.Sprintf("%d:00 - %d:59", t.Hour(), nextStep(bigInterval-time.Hour, t).Hour()),
			Data: fmt.Sprintf("%s/sH/%s", w.widgetID, t.Format(dateFormat)),
		})
		return err == nil
	})
	return err
}

func ChooseHourLayout(w *Widget) error {
	nowHour := beginningOf(time.Hour, ceilNow())
	begin := beginningOf(bigInterval, w.currDate)
	end := nextStep(bigInterval, begin)
	if begin.Before(nowHour) {
		begin = nowHour
	}

	var err error
	timeRange(begin, end, time.Hour, func(t time.Time) bool {
		err = w.addButton(tb.InlineButton{
			Text: fmt.Sprintf("%d:00 - %d:59", t.Hour(), t.Hour()),
			Data: fmt.Sprintf("%s/sS/%s", w.widgetID, t.Format(dateFormat)),
		})
		return err == nil
	})
	return err
}

func ChooseSlotLayout(w *Widget) error {
	nowSlot := beginningOf(smallInterval, ceilNow())
	begin := beginningOf(time.Hour, w.currDate)
	end := nextStep(time.Hour, begin)
	if begin.Before(nowSlot) {
		begin = nowSlot
	}

	var err error
	timeRange(begin, end, smallInterval, func(t time.Time) bool {
		err = w.addButton(tb.InlineButton{
			Text: fmt.Sprintf("%d:%02d", t.Hour(), t.Minute()),
			Data: fmt.Sprintf("%s/cS/%s", w.widgetID, t.Format(dateFormat)),
		})
		return err == nil
	})
	return err
}
