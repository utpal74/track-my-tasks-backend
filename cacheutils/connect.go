package cacheutils

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/utpal74/track-my-tasks-backend/common"
	"github.com/utpal74/track-my-tasks-backend/logger"
	"go.uber.org/zap"
)

func Connect(ctx context.Context) (*redis.Client, error) {
	log := logger.FromCtx(ctx)

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		common.FailOnError(ctx, "REDIS_URL is not set", nil)
	}

	opt, err := redis.ParseURL(redisURL)
	common.FailOnError(ctx, fmt.Sprintf("failed to parse Redis URL: %v", err), err)

	// TLS only for rediss://
	if strings.HasPrefix(redisURL, "rediss://") {
		opt.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	client := redis.NewClient(opt)

	log.Info("Redis client initialized",
		zap.String("Addr", opt.Addr),
		zap.String("Username", opt.Username),
		zap.Bool("TLS", opt.TLSConfig != nil),
	)

	// Test connection
	pingCtx, cancel := context.WithTimeout(ctx, 35*time.Second)
	defer cancel()

	pong, err := client.Ping(pingCtx).Result()
	// common.FailOnError(ctx, "Error connecting to Redis", err)

	if err != nil {
		log.Error("Redis ping failed", zap.Error(err))
	} else {
		log.Info("Redis ping response", zap.String("pong", pong))
	}

	log.Info("got response from redis client", zap.String("Redis ping response", pong))
	return client, nil
}
