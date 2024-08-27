package db

import (
	"context"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func Connect(ctx context.Context) (*mongo.Client, error) {
	// Load environment variables
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		return nil, &configError{"MONGO_URI environment variable is not set"}
	}

	// Connect to MongoDB
	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, &connectionError{err}
	}

	// Ping MongoDB to verify the connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, &pingError{err}
	}

	log.Println("Connected to MongoDB")
	return client, nil
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
