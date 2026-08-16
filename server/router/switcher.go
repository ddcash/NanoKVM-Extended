package router

import (
	"github.com/gin-gonic/gin"

	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/switcher"
)

func switcherRouter(r *gin.Engine) {
	service := switcher.NewService()

	api := r.Group("/api").Use(middleware.CheckToken())

	api.GET("/switcher", service.GetSwitcher)  // get KVM switch targets
	api.POST("/switcher", service.SetSwitcher) // set KVM switch targets
	api.POST("/switcher/press", service.PressTarget) // replay a target's hotkey
}
