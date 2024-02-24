package config

import (
	"net"
	"time"
)

type Config struct {
	LogEnviroment     string
	Address           string
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
}

func MustLoad() *Config {
	var cfg Config

	cfg = Config{
		LogEnviroment:     "local",
		Address:           net.JoinHostPort("0.0.0.0", "62801"),
		ReadTimeout:       4 * time.Second,
		WriteTimeout:      4 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return &cfg
}
