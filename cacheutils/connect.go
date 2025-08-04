package cacheutils

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
	"github.com/utpal74/track-my-tasks-backend/common"
	"github.com/utpal74/track-my-tasks-backend/logger"
	"go.uber.org/zap"
)

func Connect(ctx context.Context) (*redis.Client, error) {
	logger := logger.FromCtx(ctx)
	logger.Info("attmepting connection with redis")

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return nil, fmt.Errorf("REDIS_URL environment variable is not set")
	}

	env := os.Getenv("ENV") // "local" or "prod"
	var client *redis.Client

	serverName := os.Getenv("REDIS_TLS_SERVER_NAME")
	if serverName == "" {
		return nil, fmt.Errorf("REDIS_TLS_SERVER_NAME is not set")
	}

	if env != "production" {
		// Docker defaults: no TLS, host might be "localhost" or Docker bridge IP
		client = redis.NewClient(&redis.Options{
			Addr: redisURL,
			DB:   0,
		})
	} else {
		opt, err := redis.ParseURL(redisURL)
		common.FailOnError(ctx, fmt.Sprintf("failed to parse Redis URL: %w", err), err)

		opt.TLSConfig = &tls.Config{
			ServerName: serverName,
			InsecureSkipVerify: true,
		}
		client = redis.NewClient(opt)
	}

	

    // opt, err := redis.ParseURL(redisURL)
	// common.FailOnError(ctx, fmt.Sprintf("failed to parse Redis URL: %w", err), err)

    // opt.TLSConfig = &tls.Config{
    //     ServerName: serverName,
    //     InsecureSkipVerify: true,
    // }

    // client := redis.NewClient(opt)
	logger.Info("Pinging redis client")
    pong, err := client.Ping(ctx).Result()
	common.FailOnError(ctx, "Error connecting to Redis", err)

    logger.Info("got response from redis client", zap.String("Redis ping response:", pong))
	return  client, nil
}
