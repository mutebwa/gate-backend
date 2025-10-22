package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"gatekeeper/auth"
	"gatekeeper/db"
	"gatekeeper/middleware"
	"gatekeeper/models"
	"log"
	"net/http"
	"time"
)

type SupervisorHandler struct {
	db *db.FirestoreDB
}

func NewSupervisorHandler(firestoreDB *db.FirestoreDB) *SupervisorHandler {
	return &SupervisorHandler{
		db: firestoreDB,
	}
}

// GetEntries returns entries filtered by role
func (h *SupervisorHandler) GetEntries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		writeError(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	// Get all entries
	entries, err := h.db.GetAllEntries()
	if err != nil {
		log.Printf("❌ Failed to get entries: %v", err)
		writeError(w, "Failed to retrieve entries", http.StatusInternalServerError)
		return
	}

	// Filter based on role
	filteredEntries := filterEntriesByRole(entries, user)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entries": filteredEntries,
		"count":   len(filteredEntries),
	})
}

// ExportEntries exports entries to CSV
func (h *SupervisorHandler) ExportEntries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		writeError(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	// Get all entries
	entries, err := h.db.GetAllEntries()
	if err != nil {
		log.Printf("❌ Failed to get entries: %v", err)
		writeError(w, "Failed to retrieve entries", http.StatusInternalServerError)
		return
	}

	// Filter based on role
	filteredEntries := filterEntriesByRole(entries, user)

	// Set headers for CSV download
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("gatekeeper_entries_%s.csv", timestamp)
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// Create CSV writer
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	header := []string{
		"Record ID",
		"Entry Type",
		"Checkpoint ID",
		"Logging User ID",
		"Created At",
		"Client Timestamp",
		"Status",
		"Payload",
	}
	if err := writer.Write(header); err != nil {
		log.Printf("❌ Failed to write CSV header: %v", err)
		return
	}

	// Write data
	for _, entry := range filteredEntries {
		// Convert payload to JSON string
		payloadJSON := ""
		if entry.Payload != nil {
			if data, err := json.Marshal(entry.Payload); err == nil {
				payloadJSON = string(data)
			}
		}

		row := []string{
			entry.RecordID,
			string(entry.EntryType),
			entry.CheckpointID,
			entry.LoggingUserID,
			entry.CreatedAt.Format(time.RFC3339),
			entry.ClientTS.Format(time.RFC3339),
			string(entry.Status),
			payloadJSON,
		}
		if err := writer.Write(row); err != nil {
			log.Printf("❌ Failed to write CSV row: %v", err)
			return
		}
	}

	log.Printf("📊 CSV export by %s: %d entries", user.Username, len(filteredEntries))
}

// ResetPasswordRequest represents password reset request
type ResetPasswordRequest struct {
	UserID      string `json:"user_id"`
	NewPassword string `json:"new_password"`
}

// ResetPassword resets a user's password
func (h *SupervisorHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	supervisor, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		writeError(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" || req.NewPassword == "" {
		writeError(w, "User ID and new password are required", http.StatusBadRequest)
		return
	}

	// Validate password strength
	if err := auth.ValidatePasswordStrength(req.NewPassword); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get target user
	targetUser, err := h.db.GetUser(req.UserID)
	if err != nil {
		writeError(w, "User not found", http.StatusNotFound)
		return
	}

	// Authorization check: supervisors can only reset passwords for their managed operators
	if supervisor.Role == models.RoleSupervisor {
		canReset := false
		if supervisor.ManagedOperators != nil {
			for _, operatorID := range supervisor.ManagedOperators {
				if operatorID == req.UserID {
					canReset = true
					break
				}
			}
		}
		if !canReset {
			writeError(w, "You can only reset passwords for operators you manage", http.StatusForbidden)
			return
		}
	}
	// Admins can reset any password (already checked by middleware)

	// Hash new password
	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		log.Printf("❌ Failed to hash password: %v", err)
		writeError(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Store new password hash
	if err := h.db.StorePasswordHash(req.UserID, passwordHash); err != nil {
		log.Printf("❌ Failed to store password: %v", err)
		writeError(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	log.Printf("🔑 Password reset by %s for user: %s", supervisor.Username, targetUser.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Password reset successfully",
	})
}
