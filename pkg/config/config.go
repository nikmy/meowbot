package config

import "time"

type Config struct {
	Telegram TelegramConfig `json:"telegram"`
}

type TelegramConfig struct {
	Token        string        `json:"token,omitempty"`
	PollInterval time.Duration `json:"poll_interval" json:"poll_interval,omitempty"`
	Testing      bool          `json:"testing"`
}
