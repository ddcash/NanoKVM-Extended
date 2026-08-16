package router

import (
	"github.com/gin-gonic/gin"

	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/camera"
)

func cameraRouter(r *gin.Engine) {
	service := camera.NewService()

	// Read-only video, authenticated by the camera token rather than a session,
	// so go2rtc can pull the stream. Deliberately outside the JWT group: these
	// two routes are the only ones reachable that way, and neither can send
	// input to the target. They 404 while the feature is off.
	r.GET("/api/camera/mjpeg", service.Stream)
	r.GET("/api/camera/snapshot", service.Snapshot)

	api := r.Group("/api").Use(middleware.CheckToken())

	api.GET("/camera", service.GetConfig)  // camera access state
	api.POST("/camera", service.SetConfig) // enable or revoke
}
