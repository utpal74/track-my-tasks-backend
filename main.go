package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/utpal74/track-my-tasks-backend/cacheutils"
	"github.com/utpal74/track-my-tasks-backend/common"
	"github.com/utpal74/track-my-tasks-backend/db"
	"github.com/utpal74/track-my-tasks-backend/handlers"
	"github.com/utpal74/track-my-tasks-backend/logger"
	"github.com/utpal74/track-my-tasks-backend/routes"
	"go.mongodb.org/mongo-driver/mongo"
)

func init() {
	logger := logger.GetLogger()
	defer logger.Sync()

	if os.Getenv("ENV") != "production" {
		err := godotenv.Load(".env")
		common.FailOnError(context.TODO(), "error loading environment file", err)
		logger.Info("Successfully loaded .env file")
	} else {
		logger.Info("Running in production mode; skipping .env loading")
	}
}

func main() {
	zapLogger := logger.GetLogger()
	defer zapLogger.Sync()

	// Set the Gin mode
	mode := gin.DebugMode
	if os.Getenv("ENV") == "production" {
		mode = gin.ReleaseMode
	}
	gin.SetMode(mode)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ctx = logger.WithLogger(ctx, zapLogger)
	client, err := db.Connect(ctx)
	common.FailOnError(ctx, "error connecting DB", err)

	usersCollection := client.Database(os.Getenv("MONGO_DATABASE")).Collection("users")
	tasksCollection := client.Database(os.Getenv("MONGO_DATABASE")).Collection("tasks")

	redisClient, err := cacheutils.Connect(ctx)
	common.FailOnError(ctx, "not able to connect to redis client", err)

	taskHandler := handlers.NewTasksHandler(ctx, tasksCollection, usersCollection, redisClient)
	authHandler := handlers.NewAuthHandler(ctx, usersCollection, redisClient)
	go handleShutdown(ctx, cancel, client)
	router := setupRouter(taskHandler, authHandler)
	startServer(ctx, router)
}

func setupRouter(taskHandler *handlers.TasksHandler, authHandler *handlers.AuthHandler) *gin.Engine {
	router := gin.Default()
	allowedOrigins := strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"https://trackmytasks.net", "https://staging.trackmytasks.net", "https://www.trackmytasks.net", "https://api.trackmytasks.net"}
	}

	router.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))

	routes.SetupRoutes(router, taskHandler, authHandler)
	return router
}

func startServer(ctx context.Context, router *gin.Engine) {
	logger := logger.FromCtx(ctx)
	srv := &http.Server{
		Addr:    fmt.Sprint(":" + os.Getenv("APP_PORT")),
		Handler: router,
	}

	go func() {
		err := srv.ListenAndServe()
		common.FailOnError(ctx, "listen: ", err)
		common.FailIfServerErrored(ctx, "listen: ", err)
	}()

	// Wait for interrupt signal to gracefully shut down the server with a timeout of 10 seconds
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	logger.Info("Shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	err := srv.Shutdown(shutdownCtx)
	common.FailOnError(ctx, "Server forced to shutdown", err)
	logger.Info("Server exiting")
}

func handleShutdown(ctx context.Context, cancel context.CancelFunc, client *mongo.Client) {
	logger := logger.FromCtx(ctx)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	// Cancel the context to stop ongoing requests
	cancel()

	// Disconnect from MongoDB
	err := client.Disconnect(ctx)
	common.FailOnError(ctx, "Error while disconnecting MongoDB", err)
	logger.Info("Disconnected from MongoDB")
}
