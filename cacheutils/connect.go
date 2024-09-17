package cacheutils

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

func Connect(ctx context.Context) (*redis.Client, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDRESS"),
		Password: "",
		DB:       0,
	})

	status := redisClient.Ping(ctx)
	fmt.Println(status)

	return redisClient, nil
}
