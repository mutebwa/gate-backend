// logging.go

package main

import (
	"fmt"
	"gatekeeper/models"
	"time"
)

var mockAuditLogStore []models.AuditLog

func logAuditEvent(userID, action, details string) {
	logEntry := models.AuditLog{
		LogID:     fmt.Sprintf("log-%d", time.Now().UnixNano()),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		UserID:    userID,
		Action:    action,
		Details:   details,
	}
	mockAuditLogStore = append(mockAuditLogStore, logEntry)
	fmt.Printf("AUDIT: User '%s' performed action '%s' - Details: %s\n", userID, action, details)
}
