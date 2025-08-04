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

var (
	client *redis.Client
	redisURL, serverName string
)

// Connect returns a vlid connection with redis instance
func Connect(ctx context.Context) (*redis.Client, error) {
	logger := logger.FromCtx(ctx)
	redisURL, serverName = parseRedisConfig(ctx)
	
	// redis client setup (recommended in local development setup)
	client = redis.NewClient(&redis.Options{
		Addr: redisURL,
		DB:   0,
	})

	// redis client setup overriden in prod
	if os.Getenv("ENV") == "production" {
		logger.Info("Attempt redis connection in production mode")
		opt, err := redis.ParseURL(redisURL)
		common.FailOnError(ctx, fmt.Sprintf("failed to parse Redis URL: %v", err), err)

		opt.TLSConfig = &tls.Config{
			ServerName: serverName,
			InsecureSkipVerify: true,
		}
		client = redis.NewClient(opt)
		logger.Info("Redis client initialized", 
			zap.String("Addr", opt.Addr),
			zap.String("Username", opt.Username),
			zap.Bool("TLS", opt.TLSConfig != nil),
		)
	}

	logger.Info("Ping redis --", 
		zap.String("ENV", os.Getenv("ENV")),
		zap.String("Server name", serverName),
	)

    pong, err := client.Ping(ctx).Result()
	common.FailOnError(ctx, "Error connecting to Redis", err)

    logger.Info("got response from redis client", zap.String("Redis ping response:", pong))
	return  client, nil
}

func parseRedisConfig(ctx context.Context) (string, string) {
	redisURL = os.Getenv("REDIS_URL")
	if redisURL == "" {
		common.FailOnError(ctx, "REDIS_URL environment variable is not set", nil)
	}
	
	serverName = os.Getenv("REDIS_TLS_SERVER_NAME")
	if serverName == "" {
		common.FailOnError(ctx, "REDIS_TLS_SERVER_NAME is not set", nil)
	}

	return  redisURL, serverName
}