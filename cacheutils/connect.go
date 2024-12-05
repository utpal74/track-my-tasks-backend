package cacheutils

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/utpal74/track-my-tasks-backend/common"
	"github.com/utpal74/track-my-tasks-backend/logger"
	"go.uber.org/zap"
)

func Connect(ctx context.Context) (*redis.Client, error) {
	logger := logger.FromCtx(ctx)
	redisAddress := os.Getenv("REDIS_ADDRESS")
	if redisAddress == "" {
		return nil, fmt.Errorf("REDIS_ADDRESS environment variable is not set")
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:        redisAddress,
		Password:    "",
		DB:          0,
		DialTimeout: 30 * time.Second, // Adjust timeout as needed
		// TLSConfig: &tls.Config{
		// 	InsecureSkipVerify: true, // You can adjust based on your environment
		// },
	})

	logger.Info("Pinging redis client")
	status, err := redisClient.Ping(ctx).Result()
	common.FailOnError(ctx, "Error connecting to Redis", err)

	logger.Info("got response from redis client", zap.String("status code", status))
	return redisClient, nil
}
