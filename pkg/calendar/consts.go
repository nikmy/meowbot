package calendar

import "time"

const (
	dateFormat = time.RFC822

	bigInterval   = 6 * time.Hour
	smallInterval = 15 * time.Minute
)

var (
	// RussianMonths is a map of months names translated to Russian
	ruMonths = map[time.Month]string{
		time.January:   "Январь",
		time.February:  "Февраль",
		time.March:     "Март",
		time.April:     "Апрель",
		time.May:       "Май",
		time.June:      "Июнь",
		time.July:      "Июль",
		time.August:    "Август",
		time.September: "Сентябрь",
		time.October:   "Октябрь",
		time.November:  "Ноябрь",
		time.December:  "Декабрь",
	}

	ruWeekdays = [7]string{"Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"}
)
