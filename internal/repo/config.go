package repo

import (
	"time"
)

type BaseConfig struct {
	Interval time.Duration
}

type MongoConfig struct {
	BaseConfig

	URL     string        `yaml:"url"`
	Timeout time.Duration `yaml:"timeout"`

	Database   string `yaml:"database"`
	Collection string `yaml:"collection"`

	Auth struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"auth"`

	Pool struct {
		MinSize uint64 `yaml:"minSize"`
		MaxSize uint64 `yaml:"maxSize"`
	}
}
