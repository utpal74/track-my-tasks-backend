package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/utpal74/track-my-tasks-backend/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type TasksHandler struct {
	collections *mongo.Collection
	ctx         context.Context
}

func NewTaskHandler(ctx context.Context, collections *mongo.Collection) *TasksHandler {
	return &TasksHandler{
		collections: collections,
		ctx:         ctx,
	}
}

func (handler *TasksHandler) GetAllTasksHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cur, err := handler.collections.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	defer cur.Close(ctx)

	tasks := make([]model.Task, 0)
	for cur.Next(ctx) {
		var task model.Task
		cur.Decode(&task)
		tasks = append(tasks, task)
	}
	c.JSON(http.StatusOK, tasks)
}

func (handler *TasksHandler) NewTaskHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var task model.Task
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
	}

	task.ID = primitive.NewObjectID()
	task.Done = false

	if _, err := handler.collections.InsertOne(ctx, task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}

	c.JSON(http.StatusOK, task)
}

func (handler *TasksHandler) UpdateTaskHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	id := c.Param("id")
	var taskToBeUpdated model.Task
	if err := c.ShouldBindJSON(&taskToBeUpdated); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid id format",
		})
		return
	}

	filter := bson.M{"_id": objectId}

	fields := bson.D{}

	fields = append(fields, bson.E{Key: "id", Value: objectId})

	if taskToBeUpdated.Title != "" {
		fields = append(fields, bson.E{Key: "title", Value: taskToBeUpdated.Title})
	}

	if taskToBeUpdated.Comment != "" {
		fields = append(fields, bson.E{Key: "comment", Value: taskToBeUpdated.Comment})
	}

	if taskToBeUpdated.Done {
		fields = append(fields, bson.E{Key: "done", Value: taskToBeUpdated.Done})
	}

	update := bson.D{{Key: "$set", Value: fields}}
	result, err := handler.collections.UpdateOne(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "unable to update: " + err.Error(),
		})
		return
	}

	log.Printf("Matched %v documents and modified %v documents\n", result.MatchedCount, result.ModifiedCount)

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "No record found with the given id",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "1 record updated", "matchedCount": result.MatchedCount, "modifiedCount": result.ModifiedCount})
}

func (handler *TasksHandler) DeleteTaskHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id := c.Param("id")

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid id format :" + err.Error(),
		})
		return
	}

	if _, err := handler.collections.DeleteOne(ctx, bson.M{"_id": objectID}); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("task with id %v deleted", id)})
}

func (handler *TasksHandler) SearchTaskHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objId, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid id format " + err.Error(),
		})
	}

	var task model.Task
	if err := handler.collections.FindOne(ctx, bson.M{"_id": objId}).Decode(&task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "couldn't find task",
		})
	}

	c.JSON(http.StatusOK, task)
}
