package auth

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// Audit records security-relevant events: authentication attempts and changes
// to configuration that affects access.
//
// Scope is deliberately narrow. Only cold, low-frequency endpoints are audited.
// Nothing here is called from the WebSocket or HID paths: logging keystrokes
// would both leak whatever is typed on the target, including passwords, and add
// work to the one path that has to stay fast.
//
// Events are emitted at info level with an "audit" field so they can be
// filtered out of ordinary logs.
func Audit(c *gin.Context, event string, fields log.Fields) {
	entry := log.WithField("audit", event)

	if c != nil {
		entry = entry.WithField("ip", GetClientIP(c))
	}
	if len(fields) > 0 {
		entry = entry.WithFields(fields)
	}

	entry.Info("audit event")
}

// AuditFailure records an event that was denied or errored.
func AuditFailure(c *gin.Context, event string, reason string) {
	entry := log.WithField("audit", event).WithField("outcome", "denied")

	if c != nil {
		entry = entry.WithField("ip", GetClientIP(c))
	}

	entry.WithField("reason", reason).Warn("audit event")
}
