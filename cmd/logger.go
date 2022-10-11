package main

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func newLogger(level zapcore.Level) (*zap.SugaredLogger, error) {
	pe := zap.NewProductionEncoderConfig()
	pe.EncodeLevel = zapcore.LowercaseColorLevelEncoder
	pe.EncodeTime = zapcore.ISO8601TimeEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(pe)

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level),
	)

	l := zap.New(core)
	return l.Sugar(), nil
}
