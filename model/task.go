package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID             primitive.ObjectID `json:"id" bson:"_id"`
	Username       string             `json:"username" bson:"username"`           // Username for traditional login
	PasswordHash   string             `bson:"password,omitempty" json:"password"` // Hashed password for traditional login
	Email          string             `bson:"email,omitempty" json:"email,omitempty"`
	OAuthProviders []OAuthProvider    `json:"oauth_providers,omitempty" bson:"oauth_providers,omitempty"` // OAuth login support
	Task           []Task             `json:"task" bson:"task"`
	CreatedAt      time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
}

type Task struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	Title     string             `json:"title" bson:"title"`
	Comment   string             `json:"comment" bson:"comment"`
	Done      bool               `json:"done" bson:"done"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

type OAuthProvider struct {
	ProviderName string `json:"provider_name" bson:"provider_name"`                     // e.g., "google", "facebook"
	ProviderID   string `json:"provider_id" bson:"provider_id"`                         // Unique ID from OAuth provider
	Email        string `json:"email,omitempty" bson:"email,omitempty"`                 // Email from OAuth provider
	AccessToken  string `json:"access_token,omitempty" bson:"access_token,omitempty"`   // OAuth access token (optional)
	RefreshToken string `json:"refresh_token,omitempty" bson:"refresh_token,omitempty"` // OAuth refresh token (optional)
}
