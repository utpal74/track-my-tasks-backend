package common

import (
	"context"
	"net/http"

	"github.com/utpal74/track-my-tasks-backend/logger"
	"go.uber.org/zap"
)

// Must - a re-usable way to suppress error and returns the value
func Must(params any, err error) any {
	if err != nil {
		return err
	}

	return params
}

// Fail on error
func FailOnError(ctx context.Context, msg string, err error) {
	logIfError(ctx, msg, err, nil)
}

// Faile on closed server
func FailIfServerErrored(ctx context.Context, msg string, err error) {
	logIfError(ctx, msg, err, func(err error) bool {
		return err != http.ErrServerClosed
	})
}

func logIfError(ctx context.Context, msg string, err error, shouldLog func(error) bool) {
	logger := logger.FromCtx(ctx)
	if err != nil && (shouldLog == nil || shouldLog(err)) {
		logger.Fatal(msg, zap.Error(err))
	}
}
