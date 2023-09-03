package logger

import (
	"io"
	"os"

	"github.com/corazawaf/coraza-spoa/internal/config"
	"github.com/rs/zerolog"
)

func New(level string, file string) (*zerolog.Logger, error) {
	out, err := pathToWriter(file)
	if err != nil {
		return &zerolog.Logger{}, err
	}
	logger := zerolog.New(out).With().Timestamp().Logger()
	// set the time format to nanoseconds
	if level == "" {
		level = "info"
	}
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		return &logger, err
	}
	logger = logger.Level(lvl).Output(zerolog.ConsoleWriter{
		Out:        out,
		TimeFormat: config.DefaultTimeFormat,
	})
	return &logger, nil
}

func pathToWriter(file string) (io.Writer, error) {
	var out io.Writer
	if file == "" || file == "/dev/stdout" {
		out = os.Stdout
	} else if file == "/dev/stderr" {
		out = os.Stderr
	} else if file == "/dev/null" {
		out = io.Discard
	} else {
		f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return zerolog.Logger{}, err
		}
		out = f
	}
	return out, nil
}
