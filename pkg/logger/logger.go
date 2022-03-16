// Copyright 2022 The Corazawaf Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"time"
)

// A Level is a logging priority. Higher levels are more important.
type Level = zapcore.Level

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in
	// production.
	DebugLevel = zapcore.DebugLevel
	// InfoLevel is the default logging priority.
	InfoLevel = zapcore.InfoLevel
	// WarnLevel logs are more important than Info, but don't need individual
	// human review.
	WarnLevel = zapcore.WarnLevel
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	ErrorLevel = zapcore.ErrorLevel
	// PanicLevel logs a message, then panics.
	PanicLevel = zapcore.PanicLevel
	// FatalLevel logs a message, then calls os.Exit(1).
	FatalLevel = zapcore.FatalLevel
)

// ParseLevel parses a level based on the lower-case or all-caps ASCII
// representation of the log level. If the provided ASCII representation is
// invalid an error is returned.
//
// This is particularly useful when dealing with text input to configure log
// levels.
var ParseLevel = zapcore.ParseLevel

// Field is an alias for Field. Aliasing this type dramatically
// improves the navigability of this package's API documentation.
type Field = zap.Field

// Logger is an alias for zap.Logger. Aliasing this type dramatically
type Logger struct {
	l     *zap.Logger
	level Level
}

func (l *Logger) debug(msg string, fields ...Field) {
	l.l.Debug(msg, fields...)
}

func (l *Logger) info(msg string, fields ...Field) {
	l.l.Info(msg, fields...)
}

func (l *Logger) warn(msg string, fields ...Field) {
	l.l.Warn(msg, fields...)
}

func (l *Logger) error(msg string, fields ...Field) {
	l.l.Error(msg, fields...)
}

func (l *Logger) panic(msg string, fields ...Field) {
	l.l.Panic(msg, fields...)
}

func (l *Logger) fatal(msg string, fields ...Field) {
	l.l.Fatal(msg, fields...)
}

func (l *Logger) sync() error {
	return l.l.Sync()
}

// An Option configures a Logger.
type Option = zap.Option

var (
	// WithCaller configures the Logger to annotate each message with the filename,
	// line number, and function name of zap's caller, or not, depending on the
	// value of enabled. This is a generalized form of AddCaller.
	WithCaller = zap.WithCaller
	// AddStacktrace configures the Logger to record a stack trace for all messages at
	// or above a given level.
	AddStacktrace = zap.AddStacktrace
)

// New constructs a new Logger.
func New(writer io.Writer, level Level, ops ...Option) *Logger {
	if writer == nil {
		panic("The log writer is nil, please check it.")
	}

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = func(t time.Time, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(t.Format("2006-01-02T15:04:05.000Z0700"))
	}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg.EncoderConfig),
		zapcore.AddSync(writer),
		level,
	)
	logger := &Logger{
		l:     zap.New(core, ops...),
		level: level,
	}
	return logger
}
