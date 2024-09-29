package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/utpal74/track-my-tasks-backend/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type TasksHandler struct {
	ctx         context.Context
	mutex       sync.Mutex
	tasksColl   *mongo.Collection
	usersColl   *mongo.Collection
	redisClient *redis.Client
}

func NewTasksHandler(ctx context.Context, tasksColl *mongo.Collection, usersColl *mongo.Collection, redisClient *redis.Client) *TasksHandler {
	return &TasksHandler{
		ctx:         ctx,
		tasksColl:   tasksColl,
		usersColl:   usersColl,
		redisClient: redisClient,
	}
}

func (handler *TasksHandler) StatusHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func (handler *TasksHandler) GetAllTasksHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	username, _ := c.Get("username")
	var user model.User
	err := handler.usersColl.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	redisKey := "tasks:" + user.ID.Hex()
	cacheVal, err := handler.redisClient.Get(ctx, redisKey).Result()
	if err == redis.Nil {
		log.Printf("request to mongo DB")

		handler.mutex.Lock()
		defer handler.mutex.Unlock()

		cacheVal, err = handler.redisClient.Get(ctx, redisKey).Result()
		if err == redis.Nil {
			cur, err := handler.tasksColl.Find(ctx, bson.M{"user_id": user.ID})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			defer cur.Close(ctx)

			tasks := make([]model.Task, 0)
			for cur.Next(ctx) {
				var task model.Task
				if err := cur.Decode(&task); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode task"})
					return
				}
				tasks = append(tasks, task)
			}

			taskData, err := json.Marshal(tasks)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal tasks data"})
				return
			}

			if err := handler.redisClient.Set(ctx, redisKey, string(taskData), 10*time.Minute).Err(); err != nil {
				log.Printf("Failed to set cache for key %s: %v", redisKey, err)
			}

			c.JSON(http.StatusOK, tasks)
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	} else {
		log.Println("request from redis")
		var tasks []model.Task
		if err := json.Unmarshal([]byte(cacheVal), &tasks); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unmarshal tasks data"})
			return
		}
		c.JSON(http.StatusOK, tasks)
	}
}

func (handler *TasksHandler) NewTaskHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var task model.Task
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	username, _ := c.Get("username")
	var user model.User
	err := handler.usersColl.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	task.ID = primitive.NewObjectID()
	task.UserID = user.ID
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	// Insert the new task
	if _, err := handler.tasksColl.InsertOne(ctx, task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update the User document to include the new task ID
	update := bson.M{"$push": bson.M{"task": task}}
	_, err = handler.usersColl.UpdateOne(ctx, bson.M{"_id": user.ID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update user with new task: %v\n", err.Error())})
		return
	}

	log.Println("remove data from redis")
	redisKey := "tasks:" + user.ID.Hex()
	handler.redisClient.Del(ctx, redisKey)

	c.JSON(http.StatusOK, task)
}

func (handler *TasksHandler) UpdateTaskHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	username, _ := c.Get("username")
	var user model.User
	err := handler.usersColl.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	id := c.Param("id")
	var taskToBeUpdated model.Task
	if err := c.ShouldBindJSON(&taskToBeUpdated); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	filter := bson.M{"_id": objectId}
	updateFields := bson.M{}

	if taskToBeUpdated.Title != "" {
		updateFields["title"] = taskToBeUpdated.Title
	}

	if taskToBeUpdated.Comment != "" {
		updateFields["comment"] = taskToBeUpdated.Comment
	}

	if taskToBeUpdated.Done {
		updateFields["done"] = taskToBeUpdated.Done
	}

	updateFields["updated_at"] = time.Now()

	if len(updateFields) > 0 {
		update := bson.M{"$set": updateFields}
		result, err := handler.tasksColl.UpdateOne(ctx, filter, update)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to update: " + err.Error()})
			return
		}

		log.Printf("Matched %v documents and modified %v documents\n", result.MatchedCount, result.ModifiedCount)

		if result.MatchedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{"message": "No record found with the given id"})
			return
		}

		// Update the task in the user's Task array
		username, _ := c.Get("username")
		userUpdateFields := bson.M{}

		if taskToBeUpdated.Title != "" {
			userUpdateFields["task.$.title"] = taskToBeUpdated.Title
		}

		if taskToBeUpdated.Comment != "" {
			userUpdateFields["task.$.comment"] = taskToBeUpdated.Comment
		}

		if taskToBeUpdated.Done {
			userUpdateFields["task.$.done"] = taskToBeUpdated.Done
		}

		userUpdateFields["task.$.updated_at"] = time.Now()

		if len(userUpdateFields) > 0 {
			_, err = handler.usersColl.UpdateOne(ctx, bson.M{"username": username, "task._id": objectId},
				bson.M{"$set": userUpdateFields})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task in user collection: " + err.Error()})
				return
			}
		}

		log.Println("remove data from redis")
		redisKey := "tasks:" + user.ID.Hex()
		handler.redisClient.Del(ctx, redisKey)

		c.JSON(http.StatusOK, gin.H{"message": "1 record updated", "matchedCount": result.MatchedCount, "modifiedCount": result.ModifiedCount})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
	}
}

func (handler *TasksHandler) DeleteTaskHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	username, _ := c.Get("username")
	var user model.User
	err := handler.usersColl.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format :" + err.Error()})
		return
	}

	// Find the task to retrieve its UserID
	var task model.Task
	err = handler.tasksColl.FindOne(ctx, bson.M{"_id": objectID}).Decode(&task)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Delete the task
	_, err = handler.tasksColl.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Remove the task from the user's task list
	_, err = handler.usersColl.UpdateOne(
		ctx,
		bson.M{"_id": task.UserID},
		bson.M{"$pull": bson.M{"task": bson.M{"_id": objectID}}},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user with removed task"})
		return
	}

	log.Println("remove data from redis")
	redisKey := "tasks:" + user.ID.Hex()
	handler.redisClient.Del(ctx, redisKey)

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Task with id %v deleted", id)})
}

func (handler *TasksHandler) SearchTaskHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	username, _ := c.Get("username")
	var user model.User
	err := handler.usersColl.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	redisKey := "tasks:" + user.ID.Hex()
	cacheVal, err := handler.redisClient.Get(ctx, redisKey).Result()

	if err == redis.Nil {
		log.Println("request from DB")

		handler.mutex.Lock()
		defer handler.mutex.Unlock()

		// Check the cache again to avoid re-fetching from DB if another request already did
		cacheVal, err = handler.redisClient.Get(ctx, redisKey).Result()
		if err == redis.Nil {
			objId, err := primitive.ObjectIDFromHex(c.Param("id"))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format: " + err.Error()})
				return
			}

			var task model.Task
			if err := handler.tasksColl.FindOne(ctx, bson.M{"_id": objId}).Decode(&task); err != nil {
				if err == mongo.ErrNoDocuments {
					c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": "couldn't find task: " + err.Error()})
				return
			}

			data, err := json.Marshal(&task)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal task data"})
				return
			}

			if err := handler.redisClient.Set(ctx, redisKey, string(data), 10*time.Minute).Err(); err != nil {
				log.Printf("Failed to set cache for key %s: %v", redisKey, err)
			}

			c.JSON(http.StatusOK, task)
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	} else {
		log.Println("request from redis")
		var task model.Task
		if err := json.Unmarshal([]byte(cacheVal), &task); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unmarshal task data"})
			return
		}
		c.JSON(http.StatusOK, task)
	}
}
