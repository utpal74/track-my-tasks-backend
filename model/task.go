package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Task struct {
	ID      primitive.ObjectID `json:"id" bson:"_id"`
	Title   string             `json:"title" bson:"title"`
	Comment string             `json:"comment" bson:"comment"`
	Done    bool               `json:"done" bson:"done"`
}
