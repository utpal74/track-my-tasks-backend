package logger

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// represents production
const PROD = "production"

var (
	logger *zap.Logger
	once   sync.Once
)

// GetLogger, safely instantiate and returns only one copy of logger
func GetLogger() *zap.Logger {
	once.Do(func() {
		logger = zap.Must(zap.NewProduction())
	})

	return logger
}

type contextKey string

const (
	loggerKey contextKey = "logger"
)

// WithLogger - attach a logger to exiting context and returns context
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromCtx - returns a logger from context
func FromCtx(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(loggerKey).(*zap.Logger); ok {
		return logger
	}
	return zap.NewNop()
}
