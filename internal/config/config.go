package config

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env        string `env:"ENV" env-default:"local"`
	HTTPServer HTTPServer
	Postgres   Postgres
}

type Postgres struct {
	DatabaseURL string `env:"DATABASE_URL"`
}

type HTTPServer struct {
	Address         string        `env:"HTTP_SERVER_ADDRESS" env-default:"0.0.0.0:8080"`
	ReadTimeout     time.Duration `env:"HTTP_SERVER_READ_TIMEOUT" env-default:"5s"`
	WriteTimeout    time.Duration `env:"HTTP_SERVER_WRITE_TIMEOUT" env-default:"10s"`
	IdleTimeout     time.Duration `env:"HTTP_SERVER_IDLE_TIMEOUT" env-default:"60s"`
	ShutdownTimeout time.Duration `env:"HTTP_SERVER_SHUTDOWN_TIMEOUT" env-default:"15s"`
}

func MustLoad() *Config {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		panic("failed to read config: " + err.Error())
	}

	return &cfg
}
