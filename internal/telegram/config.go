package telegram

import "time"

type Config struct {
	Token        string        `yaml:"token"`
	PollInterval time.Duration `yaml:"pollInterval"`
}
