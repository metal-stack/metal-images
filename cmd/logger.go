package main

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func newLogger(level zapcore.Level) (*zap.SugaredLogger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(level)
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zlog, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	return zlog.Sugar(), nil
}
