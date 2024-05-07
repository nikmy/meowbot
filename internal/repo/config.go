package repo

import (
	"time"
)

type dbKind string

const (
	mongoDB dbKind = "mongo"
)

type Config struct {
	MongoCfg *MongoConfig `yaml:"mongo"`
}

type MongoConfig struct {
	Interval time.Duration `yaml:"interval"`

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
