package main

import (
	"flag"
	"github.com/nikmy/meowbot/internal/repo"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/nikmy/meowbot/internal/telegram"
	"github.com/nikmy/meowbot/pkg/environment"
	"github.com/nikmy/meowbot/pkg/errors"
)

type Config struct {
	Environment environment.Env `yaml:"Environment"`
	Telegram    telegram.Config `yaml:"Telegram"`
	Mongo repo.MongoConfig `yaml:"Mongo"`
}

func loadConfig() (*Config, error) {
	path, err := filepath.Abs("config.yaml")
	if err != nil {
		return nil, errors.WrapFail(err, "build path to config")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.WrapFail(err, "read \"config.json\"")
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, errors.WrapFail(err, "parse yaml")
	}

	if envFromFlags := getEnvFromFlags(); envFromFlags != nil {
		cfg.Environment = *envFromFlags
	}

	return &cfg, nil
}

func getEnvFromFlags() *environment.Env {
	raw := flag.String("env", "", "environment (dev, prod)")
	flag.Parse()
	if raw == nil {
		return nil
	}

	env := environment.FromString(*raw)
	return &env
}
