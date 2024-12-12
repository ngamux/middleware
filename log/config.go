package log

import (
	"log/slog"
)

type Config struct {
	Handler slog.Handler
}

func configDefault() Config {
	return Config{}
}
