package router

import (
	"github.com/gin-gonic/gin"

	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/mqtt"
)

func mqttRouter(r *gin.Engine) {
	service := mqtt.NewService()

	api := r.Group("/api").Use(middleware.CheckToken())

	api.GET("/mqtt/config", service.GetConfig)  // get mqtt configuration
	api.POST("/mqtt/config", service.SetConfig) // set mqtt configuration
	api.POST("/mqtt/publish", service.Publish)  // publish a configured command
}
