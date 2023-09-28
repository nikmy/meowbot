package calendar

import (
	"github.com/nikmy/meowbot/pkg/tools/builder"
	"time"
)

func NewCalendarWidget(wid string, binder, ignoreBinder inlineBinder, lang string, currDate time.Time) (*Widget, error) {
	return builder.New[Widget]().
		Use(AsCurrentDate(currDate)).
		Use(AsLanguage(lang)).
		Use(AsBinder(binder)).
		Use(AsIgnore(ignoreBinder)).
		Use(AsID(wid)).
		MaybeUse(ReturnButton).
		MaybeUse(YearMonthButton).
		MaybeUse(WeekdaysLayout).
		MaybeUse(ChooseDayLayout).
		Get()
}
