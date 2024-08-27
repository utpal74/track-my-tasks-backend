package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/utpal74/track-my-tasks-backend/handlers"
)

func SetupRoutes(router *gin.Engine, handler *handlers.TasksHandler) {
	router.GET("/tasks", handler.GetAllTasksHandler)
	router.POST("/tasks/create", handler.NewTaskHandler)
	router.PUT("/tasks/update/:id", handler.UpdateTaskHandler)
	router.DELETE("/tasks/delete/:id", handler.DeleteTaskHandler)
	router.GET("/tasks/search/:id", handler.SearchTaskHandler)
}
