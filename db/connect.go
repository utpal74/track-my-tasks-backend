package db

import (
	"context"
	"os"

	"github.com/utpal74/track-my-tasks-backend/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
)

func Connect(ctx context.Context) (*mongo.Client, error) {
	logger := logger.FromCtx(ctx)

	// Load environment variables
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		return nil, &configError{"MONGO_URI environment variable is not set"}
	}

	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(ctx, clientOptions)
	checkAndThrowError(client, err, &connectionError{err})

	logger.Info("pinging mongo db")
	err = client.Ping(ctx, readpref.Primary())
	checkAndThrowError(nil, err, &pingError{err})
	logger.Info("mongo db ping successful", zap.String("database", os.Getenv("MONGO_DATABASE")))
	return client, nil
}

func checkAndThrowError(params any, err error, errToThrow error) (any, error) {
	if err != nil {
		return nil, errToThrow
	}
	return params, err
}

type configError struct {
	message string
}

func (e *configError) Error() string {
	return e.message
}

type connectionError struct {
	err error
}

func (e *connectionError) Error() string {
	return "Failed to connect to MongoDB: " + e.err.Error()
}

type pingError struct {
	err error
}

func (e *pingError) Error() string {
	return "Failed to ping MongoDB: " + e.err.Error()
}
