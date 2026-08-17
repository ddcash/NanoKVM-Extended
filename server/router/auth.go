package router

import (
	"github.com/gin-gonic/gin"

	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/auth"
)

func authRouter(r *gin.Engine) {
	service := auth.NewService()

	r.POST("/api/auth/login", service.Login) // login

	api := r.Group("/api").Use(middleware.CheckToken())

	api.GET("/auth/password", service.IsPasswordUpdated) // is password updated
	api.GET("/auth/account", service.GetAccount)         // get account
	api.POST("/auth/password", service.ChangePassword)   // change password
	api.POST("/auth/logout", service.Logout)             // logout

	api.GET("/auth/sessions", service.GetSessions)       // active sessions
	api.POST("/auth/sessions/revoke", service.RevokeSession)

	api.GET("/auth/totp", service.GetTotp)               // two-factor state
	api.POST("/auth/totp/setup", service.SetupTotp)      // generate a secret
	api.POST("/auth/totp/enable", service.EnableTotp)    // confirm code, enable
	api.POST("/auth/totp/disable", service.DisableTotp)  // disable (password + code)
}
