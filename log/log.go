// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package log

import (
	"io"
	"os"

	"github.com/corazawaf/coraza/v3/types"
	"github.com/rs/zerolog"
)

var Logger = zerolog.New(os.Stderr).Level(zerolog.InfoLevel).With().Timestamp().Logger()

var WafErrorCallback = func(mr types.MatchedRule) {
	Logger.
		WithLevel(convert(mr.Rule().Severity())).
		Msg(mr.ErrorLog())
}

// InitLogging initializes the logging.
func InitLogging(file, level string) {
	if level == "" && file == "" {
		Debug().Msg("Nothing to configure, using standard logger")
		return
	}

	logger := Logger

	if file != "" {
		out, err := resolveLogPath(file)
		if err != nil {
			Error().Err(err).Msg("Can't open log file, using standard")
		} else {
			logger = logger.Output(out)
		}
	}

	currentLevel := Logger.GetLevel()
	targetLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		Error().Err(err).Msgf("Can't parse log level, using %v log level", currentLevel)

	} else if targetLevel < currentLevel {
		Debug().Msgf("Setting up %v log level", targetLevel)
		logger = logger.Level(targetLevel)
	}

	Logger = logger
}

func SetDebug(debug bool) {
	if debug && Logger.GetLevel() != zerolog.DebugLevel {
		Logger = Logger.Level(zerolog.DebugLevel)
		Debug().Msgf("Using %v log level", zerolog.DebugLevel)
	}
}

func Trace() *zerolog.Event {
	return Logger.Trace()
}

func Debug() *zerolog.Event {
	return Logger.Debug()
}

func Info() *zerolog.Event {
	return Logger.Info()
}

func Warn() *zerolog.Event {
	return Logger.Warn()
}

func Error() *zerolog.Event {
	return Logger.Error()
}

func Fatal() *zerolog.Event {
	return Logger.Fatal()
}
func Panic() *zerolog.Event {
	return Logger.Panic()
}

func resolveLogPath(path string) (io.Writer, error) {
	switch path {
	case "":
		return io.Discard, nil

	case "/dev/stdout":
		return os.Stdout, nil

	case "/dev/stderr":
		return os.Stderr, nil

	default:
		return os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	}
}

func convert(severity types.RuleSeverity) zerolog.Level {
	switch severity {
	case types.RuleSeverityEmergency, types.RuleSeverityAlert, types.RuleSeverityCritical, types.RuleSeverityError:
		return zerolog.ErrorLevel

	case types.RuleSeverityWarning:
		return zerolog.WarnLevel

	case types.RuleSeverityNotice, types.RuleSeverityInfo:
		return zerolog.InfoLevel

	case types.RuleSeverityDebug:
		return zerolog.DebugLevel
	}
	return zerolog.Disabled
}
