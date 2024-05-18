package telegram

import "time"

type Config struct {
	BotConfig           `yaml:"bot"`
	NotificationsConfig `yaml:"notifications"`
	TimeZoneConfig      `yaml:"time_zone"`
}

type BotConfig struct {
	Token        string        `yaml:"token"`
	PollInterval time.Duration `yaml:"pollInterval"`
}

type NotificationsConfig struct {
	NotifyBefore []time.Duration `yaml:"notifyBefore"`
	NotifyPeriod time.Duration   `yaml:"notifyPeriod"`
}

type TimeZoneConfig struct {
	Name    string        `yaml:"name"`
	UTCDiff time.Duration `yaml:"utcDiff"`
}
