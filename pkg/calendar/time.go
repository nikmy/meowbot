package calendar

import (
	"time"
)

func timeRange(begin time.Time, end time.Time, step time.Duration, iterFunc func(it time.Time) bool) {
	for it := begin; it.Before(end); it = nextStep(step, it) {
		if !iterFunc(it) {
			break
		}
	}
}

func beginningOfYear(ts time.Time) time.Time {
	y, _, _ := ts.Date()
	return time.Date(y, 1, 1, 0, 0, 0, 0, ts.Location())
}

func beginningOfMonth(ts time.Time) time.Time {
	y, m, _ := ts.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, ts.Location())
}

func beginningOfDay(ts time.Time) time.Time {
	y, m, d := ts.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, ts.Location())
}

func beginningOf(interval time.Duration, ts time.Time) time.Time {
	y, M, d := ts.Date()
	m := ts.Hour()*60 + ts.Minute()

	mInterval := int(interval.Minutes())
	m -= m % mInterval

	return time.Date(y, M, d, 0, m, 0, 0, ts.Location())
}

func ceilNow() time.Time {
	return time.Now().Add(smallInterval)
}

func beginningOfTomorrow(ts time.Time) time.Time {
	return time.Date(ts.Year(), ts.Month(), ts.Day()+1, 0, 0, 0, 0, ts.Location())
}

func nextStep(step time.Duration, ts time.Time) time.Time {
	return ts.Add(step)
}
