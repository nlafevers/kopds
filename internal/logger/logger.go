package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// New initializes a new logger.
func New(level string, json bool) zerolog.Logger {
	var log zerolog.Logger

	if json {
		log = zerolog.New(os.Stderr).With().Timestamp().Logger()
	} else {
		output := zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		}
		log = zerolog.New(output).With().Timestamp().Logger()
	}

	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	return log.Level(lvl)
}
