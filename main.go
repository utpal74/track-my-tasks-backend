package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/utpal74/track-my-tasks-backend/cacheutils"
	"github.com/utpal74/track-my-tasks-backend/db"
	"github.com/utpal74/track-my-tasks-backend/handlers"
	"github.com/utpal74/track-my-tasks-backend/routes"
	"go.mongodb.org/mongo-driver/mongo"
)

func init() {
	if err := godotenv.Load(".env"); err != nil {
		log.Fatalf("error loading environment : %v", err)
		return
	}
	log.Println("Successfully loaded .env file")
}

func main() {
	// Create a new context with a timeout for connecting to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// connect to mongo db
	client, err := db.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize the collection
	usersCollection := client.Database(os.Getenv("MONGO_DATABASE")).Collection("users")
	tasksCollection := client.Database(os.Getenv("MONGO_DATABASE")).Collection("tasks")

	// Initialize redis
	redisClient, err := cacheutils.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize taskHandler
	taskHandler := handlers.NewTasksHandler(ctx, tasksCollection, usersCollection, redisClient)
	authHandler := handlers.NewAuthHandler(ctx, usersCollection, redisClient)

	// Ensure clean up during shutdown
	go handleShutdown(cancel, client)

	// Set up the Gin router with CORS middleware
	router := setupRouter(taskHandler, authHandler)

	// Start the server
	startServer(router)
}

func setupRouter(taskHandler *handlers.TasksHandler, authHandler *handlers.AuthHandler) *gin.Engine {
	router := gin.Default()

	// Configure CORS dynamically for different environments
	allowedOrigins := []string{"http://localhost:5173"}
	router.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))

	// Set up routes
	routes.SetupRoutes(router, taskHandler, authHandler)

	return router
}

func startServer(router *gin.Engine) {
	srv := &http.Server{
		Addr:    ":8082",
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server with a timeout of 10 seconds
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}

func handleShutdown(cancel context.CancelFunc, client *mongo.Client) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	// Cancel the context to stop ongoing requests
	cancel()

	// Disconnect from MongoDB
	if err := client.Disconnect(context.Background()); err != nil {
		log.Fatal("Error while disconnecting MongoDB: ", err)
	}

	log.Println("Disconnected from MongoDB")
}
