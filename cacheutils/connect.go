package cacheutils

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

func Connect(ctx context.Context) (*redis.Client, error) {
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

	status, err := redisClient.Ping(ctx).Result()
	if err != nil {
		fmt.Println("Error connecting to Redis:", err)
		return nil, err
	}

	fmt.Println("Connected to Redis:", status)
	return redisClient, nil
}
