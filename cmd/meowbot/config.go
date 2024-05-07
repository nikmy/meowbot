package main

import (
	"flag"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/nikmy/meowbot/internal/repo"
	"github.com/nikmy/meowbot/internal/telegram"
	"github.com/nikmy/meowbot/pkg/environment"
	"github.com/nikmy/meowbot/pkg/errors"
)

type Config struct {
	Environment  environment.Env `yaml:"Environment"`
	Telegram     telegram.Config `yaml:"Telegram"`
	InterviewsDB repo.Config     `yaml:"InterviewsDB"`
	UsersDB      repo.Config     `yaml:"UsersDB"`
}

func loadConfig() (*Config, error) {
	loadFlags()

	path, err := filepath.Abs(*cfgFile)
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

	if envFromFlags != nil {
		cfg.Environment = *envFromFlags
	}

	return &cfg, nil
}

var (
	envFromFlags *environment.Env
	envRaw       = flag.String("env", "dev", "environment (dev, prod)")
	cfgFile      = flag.String("config", "config.yaml", "path to a config file")
)

func loadFlags() {
	flag.Parse()

	if envRaw != nil {
		envFromFlags = new(environment.Env)
		*envFromFlags = environment.FromString(*envRaw)
	}
}
