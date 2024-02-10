package telegram

import "time"

type Config struct {
	Token        string        `yaml:"token,omitempty"`
	PollInterval time.Duration `yaml:"poll_interval"`
}
