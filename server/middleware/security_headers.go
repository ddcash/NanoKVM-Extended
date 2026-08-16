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
// There is deliberately no script-src or style-src policy. The frontend and its
// dependencies inject inline <script> and <style> at runtime, so any policy
// tight enough to be worth having would have to be built from per-build hashes
// or a nonce; a blanket 'unsafe-inline' would allow exactly what script-src
// exists to prevent, while still breaking on the next dependency that inlines
// something. The headers kept below are the ones that protect this device
// without depending on how the bundle is emitted.
//
// HSTS is also omitted: the device presents a self-signed certificate on a
// private address, and an HSTS entry pinned in the browser for a host that
// later serves plain HTTP locks the UI out with no obvious way back.
func SecurityHeaders(isTLS bool) gin.HandlerFunc {
	options := secure.Options{
		// Clickjacking protection, which is the realistic web threat to a
		// device that can control a machine's keyboard and mouse.
		FrameDeny:          true,
		ContentTypeNosniff: true,
		ReferrerPolicy:     "same-origin",
		// frame-ancestors is the modern equivalent of FrameDeny and, unlike
		// script-src, cannot be tripped by inlined bundle output.
		ContentSecurityPolicy: "frame-ancestors 'none'; base-uri 'self'; form-action 'self'",
		IsDevelopment:         !isTLS,
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
