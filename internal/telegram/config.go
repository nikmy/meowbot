package telegram

import "time"

type Config struct {
	Token        string        `yaml:"token"`
	PollInterval time.Duration `yaml:"pollInterval"`

	NotifyBefore []time.Duration `yaml:"notifyBefore"`
	NotifyPeriod time.Duration   `yaml:"notifyPeriod"`
}
