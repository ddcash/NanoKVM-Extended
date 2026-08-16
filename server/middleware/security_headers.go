package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/unrolled/secure"
)

// SecurityHeaders sets defensive response headers.
//
// This runs once per HTTP request, at the point the request enters gin. The
// WebSocket video and HID streams are unaffected: gin middleware runs on the
// upgrade handshake only, never on the frames that follow, so nothing here sits
// on the latency-sensitive path.
//
// HSTS is deliberately not enabled by default. The device presents a
// self-signed certificate on a private address, and an HSTS entry pinned in the
// browser for a host that later serves plain HTTP locks the UI out with no
// obvious way back.
func SecurityHeaders(isTLS bool) gin.HandlerFunc {
	options := secure.Options{
		FrameDeny:          true,
		ContentTypeNosniff: true,
		ReferrerPolicy:     "same-origin",
		// The frontend is served from the same origin as the API. Inline styles
		// and blobs are required: antd injects <style> at runtime, and the H.264
		// player builds its stream from blob URLs.
		ContentSecurityPolicy: "default-src 'self'; " +
			"img-src 'self' data: blob:; " +
			"media-src 'self' data: blob:; " +
			"style-src 'self' 'unsafe-inline'; " +
			"script-src 'self' 'wasm-unsafe-eval'; " +
			"connect-src 'self' ws: wss:; " +
			"worker-src 'self' blob:; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self'",
		IsDevelopment: !isTLS,
	}

	secureMiddleware := secure.New(options)

	return func(c *gin.Context) {
		if err := secureMiddleware.Process(c.Writer, c.Request); err != nil {
			c.Abort()
			return
		}
		c.Next()
	}
}
