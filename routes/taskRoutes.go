package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/utpal74/track-my-tasks-backend/handlers"
)

func SetupRoutes(router *gin.Engine, taskHandler *handlers.TasksHandler, authHandler *handlers.AuthHandler) {
	router.GET("/", taskHandler.StatusHandler)
	router.POST("/signin", authHandler.SignInHandler)
	router.POST("/signup", authHandler.SignUpHandler)

	auth := router.Group("/")
	auth.Use(authHandler.AuthMiddleware())

	// authenticated api request
	{
		auth.GET("/tasks", taskHandler.GetAllTasksHandler)
		auth.POST("/tasks/create", taskHandler.NewTaskHandler)
		auth.PUT("/tasks/update/:id", taskHandler.UpdateTaskHandler)
		auth.DELETE("/tasks/delete/:id", taskHandler.DeleteTaskHandler)
		auth.GET("/tasks/search/:id", taskHandler.SearchTaskHandler)
		auth.POST("/refresh", authHandler.RefreshHandler)
	}
}
