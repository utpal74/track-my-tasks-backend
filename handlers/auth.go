package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	redis "github.com/redis/go-redis/v9"
	"github.com/rs/xid"
	"github.com/utpal74/track-my-tasks-backend/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuthHandler struct {
	ctx         context.Context
	collections *mongo.Collection
	redisClient *redis.Client
}

func NewAuthHandler(ctx context.Context, collections *mongo.Collection, redisClient *redis.Client) *AuthHandler {
	return &AuthHandler{
		ctx:         ctx,
		collections: collections,
		redisClient: redisClient,
	}
}

func (handler *AuthHandler) SignUpHandler(c *gin.Context) {
	// Create a new context with a timeout for this request
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	// Check if the username or email already exists
	filter := bson.M{"$or": []bson.M{
		{"username": user.Username},
		{"email": user.Email},
	}}
	existingUser := handler.collections.FindOne(ctx, filter)
	if existingUser.Err() == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username or Email already exists"})
		return
	}

	// If this is a traditional sign-up (password-based), hash the password
	if user.PasswordHash != "" {
		h := sha256.New()
		h.Write([]byte(user.PasswordHash))
		hashedPwd := hex.EncodeToString(h.Sum(nil))
		user.PasswordHash = hashedPwd
	} else if len(user.OAuthProviders) == 0 {
		// If there's no password and no OAuth providers, it's an invalid sign-up
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password or OAuth provider required"})
		return
	}

	// Prepare user object
	user.ID = primitive.NewObjectID()
	user.Task = []model.Task{}
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	// Insert the new user into MongoDB
	_, err := handler.collections.InsertOne(ctx, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Could not create user: %v", err.Error())})
		return
	}

	// Respond with success
	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
}

// SignInHandler - Sign in user and store session in Redis
func (handler *AuthHandler) SignInHandler(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create a new context with a timeout for this request
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Hash the incoming password
	h := sha256.New()
	h.Write([]byte(user.PasswordHash))
	hashedPwd := hex.EncodeToString(h.Sum(nil))

	// Find the user in MongoDB
	cur := handler.collections.FindOne(ctx, bson.M{
		"username":      user.Username,
		"password_hash": hashedPwd,
	})

	if cur.Err() != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Invalid credentials: %v", cur.Err())})
		return
	}

	// Generate a new session token
	sessionToken := xid.New().String()

	// Store the session token in Redis, valid for 10 minutes
	err := handler.redisClient.Set(ctx, sessionToken, user.Username, 10*time.Minute).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save session in Redis"})
		return
	}

	// Respond with success
	c.JSON(http.StatusOK, gin.H{"message": "User signed in", "token": sessionToken})
}

// RefreshHandler - Refresh the session token and extend its lifetime in Redis
func (handler *AuthHandler) RefreshHandler(c *gin.Context) {
	// Get the old session token from the request header or cookie
	oldToken := c.Request.Header.Get("Authorization")
	if oldToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No active session"})
		return
	}

	// Create a new context with a timeout for this request
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if the session exists in Redis
	username, err := handler.redisClient.Get(ctx, oldToken).Result()
	if err == redis.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired or not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking session"})
		return
	}

	// Generate a new session token
	newToken := xid.New().String()

	// Delete the old session and set the new one with an extended expiration
	_, err = handler.redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, oldToken)
		pipe.Set(ctx, newToken, username, 10*time.Minute)
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not refresh session"})
		return
	}

	// Respond with the new token
	c.JSON(http.StatusOK, gin.H{"message": "Session refreshed", "new_token": newToken})
}

// AuthMiddleware - Middleware to protect routes, checking for valid session in Redis
func (handler *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the session token from the Authorization header
		sessionToken := c.Request.Header.Get("Authorization")
		if sessionToken == "" {
			c.JSON(http.StatusForbidden, gin.H{"message": "No session token provided"})
			c.Abort()
			return
		}

		// Create a new context with a timeout for this request
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Check if the session token exists in Redis
		username, err := handler.redisClient.Get(ctx, sessionToken).Result()
		if err == redis.Nil {
			c.JSON(http.StatusForbidden, gin.H{"message": "Invalid or expired session token"})
			c.Abort()
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error checking session token: %v", err.Error())})
			c.Abort()
			return
		}

		// Store the username in the context for later use
		c.Set("username", username)

		// Continue to the next handler
		c.Next()
	}
}
