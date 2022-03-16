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
	"gopkg.in/natefinch/lumberjack.v2"
	"time"
)

// LevelEnablerFunc is a function type that returns a bool indicating whether logging beyond a given Level should be enabled.
type LevelEnablerFunc func(Level) bool

// RotateOptions used to configure the rotation of the log file.
type RotateOptions struct {
	MaxSize    int
	MaxAge     int
	MaxBackups int
	Compress   bool
}

// TeeOption used to configure the tee logger.
type TeeOption struct {
	Filename string
	ROpts    RotateOptions
	Lef      LevelEnablerFunc
}

// NewTeeWithRotate creates a new tee logger that contains multiple loggers with the rotation of the log function.
func NewTeeWithRotate(tops []TeeOption, opts ...Option) *Logger {
	var cores []zapcore.Core
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = func(t time.Time, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(t.Format("2006-01-02T15:04:05.000Z0700"))
	}
	for _, top := range tops {
		w := zapcore.AddSync(&lumberjack.Logger{
			Filename:   top.Filename,
			MaxSize:    top.ROpts.MaxSize,
			MaxAge:     top.ROpts.MaxAge,
			MaxBackups: top.ROpts.MaxBackups,
			Compress:   top.ROpts.Compress,
		})

		core := zapcore.NewCore(
			zapcore.NewConsoleEncoder(cfg.EncoderConfig),
			zapcore.AddSync(w),
			zap.LevelEnablerFunc(func(level zapcore.Level) bool {
				return top.Lef(level)
			}),
		)
		cores = append(cores, core)
	}
	return &Logger{
		l: zap.New(zapcore.NewTee(cores...), opts...),
	}
}
