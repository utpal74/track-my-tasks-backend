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

// redis key
var redisKey string = "tasks"

type TasksHandler struct {
	ctx         context.Context
	mutex       sync.Mutex
	collections *mongo.Collection
	redisClient *redis.Client
}

func NewTaskHandler(ctx context.Context, collections *mongo.Collection, redisClient *redis.Client) *TasksHandler {
	return &TasksHandler{
		ctx:         ctx,
		collections: collections,
		redisClient: redisClient,
	}
}

func (handler *TasksHandler) GetAllTasksHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cacheVal, err := handler.redisClient.Get(ctx, redisKey).Result()
	if err == redis.Nil {
		log.Printf("request to mongo DB")

		handler.mutex.Lock()
		defer handler.mutex.Unlock()

		cacheVal, err = handler.redisClient.Get(ctx, redisKey).Result()
		if err == redis.Nil {
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

			taskData, _ := json.Marshal(tasks)
			if err := handler.redisClient.Set(ctx, redisKey, string(taskData), time.Minute*10).Err(); err != nil {
				log.Printf("Failed to set cache for key %s: %v", redisKey, err)
			}

			c.JSON(http.StatusOK, tasks)
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	} else {
		log.Println("request to redis")
		var tasks []model.Task
		json.Unmarshal([]byte(cacheVal), &tasks)
		c.JSON(http.StatusOK, tasks)
	}
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

	log.Println("remove data from redis")
	handler.redisClient.Del(ctx, redisKey)

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

	log.Println("remove data from redis")
	handler.redisClient.Del(ctx, redisKey)

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

	log.Println("remove data from redis")
	handler.redisClient.Del(ctx, redisKey)

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("task with id %v deleted", id)})
}

func (handler *TasksHandler) SearchTaskHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	redisKey := "task:" + c.Param("id")
	cacheVal, err := handler.redisClient.Get(ctx, redisKey).Result()

	if err == redis.Nil {
		log.Println("request from DB")

		handler.mutex.Lock()
		defer handler.mutex.Unlock()

		cacheVal, err = handler.redisClient.Get(ctx, redisKey).Result()
		if err == redis.Nil {
			objId, err := primitive.ObjectIDFromHex(c.Param("id"))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid id format " + err.Error(),
				})
				return
			}

			var task model.Task
			if err := handler.collections.FindOne(ctx, bson.M{"_id": objId}).Decode(&task); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "couldn't find task",
				})
				return
			}

			data, err := json.Marshal(&task)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal task data"})
				return
			}

			if err := handler.redisClient.Set(ctx, redisKey, string(data), time.Minute*10).Err(); err != nil {
				log.Printf("Failed to set cache for key %s: %v", redisKey, err)
			}

			c.JSON(http.StatusOK, task)
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	} else {
		log.Println("request from redis")
		var task model.Task
		json.Unmarshal([]byte(cacheVal), &task)
		c.JSON(http.StatusOK, task)
	}
}
