package repo

import (
	"time"
)

type DataSource string

type MongoConfig struct {
	Interval time.Duration `yaml:"interval"`

	URL     string        `yaml:"url"`
	Timeout time.Duration `yaml:"timeout"`

	Database string `yaml:"database"`

	Auth struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"auth"`

	Pool struct {
		MinSize uint64 `yaml:"minSize"`
		MaxSize uint64 `yaml:"maxSize"`
	}
}
