package api

import "time"

type Config struct {
	Proxy struct {
		Header  string   `yaml:"header"`
		Trusted []string `yaml:"trusted"`
	} `yaml:"proxy"`

	HTTP struct {
		Addr         string        `yaml:"addr"`
		ReadTimeout  time.Duration `yaml:"read_timeout"`
		WriteTimeout time.Duration `yaml:"write_timeout"`
		IdleTimeout  time.Duration `yaml:"idle_timeout"`
	} `yaml:"http"`
}
