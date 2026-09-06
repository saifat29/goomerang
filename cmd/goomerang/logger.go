package main

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/saifat29/goomerang/config"
)

func setupLogger(cfg *config.Config) {
	zerolog.TimestampFunc = func() time.Time {
		return time.Now().UTC()
	}

	level, err := zerolog.ParseLevel(cfg.Logging.Level)
	if err != nil {
		log.Warn().Err(err).Str("level", cfg.Logging.Level).Msg("invalid log level, falling back to error")
		level = zerolog.ErrorLevel
	}
	zerolog.SetGlobalLevel(level)

	var output io.Writer = os.Stderr
	if cfg.Logging.Format == "console" {
		output = zerolog.ConsoleWriter{
			Out:          os.Stderr,
			TimeFormat:   time.RFC3339,
			TimeLocation: time.UTC,
		}
	}

	logger := zerolog.New(output).With().Timestamp().Logger()
	if level <= zerolog.DebugLevel {
		// Add file and line number to log entries when in debug mode.
		logger = logger.With().Caller().Logger()
	}
	log.Logger = logger
}
