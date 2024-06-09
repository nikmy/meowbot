package telegram

import "time"

type timeProvider interface {
	Now() time.Time
	NowMillis() int64
	UTCDiff() time.Duration
	ZoneName() string
}

type stdTime struct{
	utcDiff time.Duration
	zoneName string
}

func (s stdTime) ZoneName() string {
	return s.zoneName
}

func (s stdTime) UTCDiff() time.Duration {
	return s.utcDiff
}

func (s stdTime) Now() time.Time {
	return time.Now().UTC().Add(s.utcDiff)
}

func (s stdTime) NowMillis() int64 {
	return time.Now().UTC().Add(s.utcDiff).UnixMilli()
}


