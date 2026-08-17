package router

import (
	"github.com/gin-gonic/gin"

	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/actions"
)

func actionsRouter(r *gin.Engine) {
	service := actions.NewService()

	api := r.Group("/api").Use(middleware.CheckToken())

	api.GET("/actions", service.GetActions)      // custom GPIO and command actions
	api.POST("/actions", service.SetActions)     // save them
	api.POST("/actions/run", service.RunAction)  // trigger one
}
