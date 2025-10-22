package handlers

import (
	"encoding/json"
	"gatekeeper/db"
	"gatekeeper/middleware"
	"gatekeeper/models"
	"log"
	"net/http"
	"time"
)

type SyncHandler struct {
	db *db.FirestoreDB
}

func NewSyncHandler(firestoreDB *db.FirestoreDB) *SyncHandler {
	return &SyncHandler{
		db: firestoreDB,
	}
}

// SyncPushRequest represents the request body for sync push
type SyncPushRequest struct {
	Entries []models.Entry `json:"entries"`
}

// SyncPushResponse represents the response for sync push
type SyncPushResponse struct {
	Success      bool     `json:"success"`
	Accepted     int      `json:"accepted"`
	Rejected     int      `json:"rejected"`
	RejectedIDs  []string `json:"rejected_ids,omitempty"`
	Message      string   `json:"message"`
}

// SyncPullResponse represents the response for sync pull
type SyncPullResponse struct {
	Entries []models.Entry `json:"entries"`
	Count   int            `json:"count"`
}

// Push handles syncing entries from client to server
func (h *SyncHandler) Push(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		writeError(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	var req SyncPushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	accepted := 0
	rejected := 0
	var rejectedIDs []string

	for _, entry := range req.Entries {
		// Validate entry belongs to user (security check)
		if entry.LoggingUserID != user.UserID {
			log.Printf("⚠️  User %s attempted to push entry for user %s", user.Username, entry.LoggingUserID)
			rejected++
			rejectedIDs = append(rejectedIDs, entry.RecordID)
			continue
		}

		// Validate checkpoint access for gate operators
		if user.Role == models.RoleGateOperator {
			hasAccess := false
			for _, cp := range user.AllowedCheckpoints {
				if cp == entry.CheckpointID {
					hasAccess = true
					break
				}
			}
			if !hasAccess {
				log.Printf("⚠️  User %s attempted to push entry for unauthorized checkpoint %s", user.Username, entry.CheckpointID)
				rejected++
				rejectedIDs = append(rejectedIDs, entry.RecordID)
				continue
			}
		}

		// Create entry in Firestore
		if err := h.db.CreateEntry(&entry); err != nil {
			log.Printf("❌ Failed to create entry %s: %v", entry.RecordID, err)
			rejected++
			rejectedIDs = append(rejectedIDs, entry.RecordID)
			continue
		}

		accepted++
	}

	log.Printf("📤 Sync push from %s: %d accepted, %d rejected", user.Username, accepted, rejected)

	response := SyncPushResponse{
		Success:     rejected == 0,
		Accepted:    accepted,
		Rejected:    rejected,
		RejectedIDs: rejectedIDs,
		Message:     "Sync completed",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Pull handles syncing entries from server to client
func (h *SyncHandler) Pull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		writeError(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	sinceParam := query.Get("since")

	var entries []models.Entry
	var err error

	// If 'since' parameter is provided, get entries after that timestamp
	if sinceParam != "" {
		sinceTime, parseErr := time.Parse(time.RFC3339, sinceParam)
		if parseErr != nil {
			writeError(w, "Invalid 'since' parameter format. Use RFC3339", http.StatusBadRequest)
			return
		}
		entries, err = h.db.GetEntriesSince(sinceTime)
	} else {
		// Get all entries
		entries, err = h.db.GetAllEntries()
	}

	if err != nil {
		log.Printf("❌ Failed to get entries: %v", err)
		writeError(w, "Failed to retrieve entries", http.StatusInternalServerError)
		return
	}

	// Filter entries based on user role
	filteredEntries := filterEntriesByRole(entries, user)

	log.Printf("📥 Sync pull for %s: %d entries", user.Username, len(filteredEntries))

	response := SyncPullResponse{
		Entries: filteredEntries,
		Count:   len(filteredEntries),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// filterEntriesByRole filters entries based on user role and permissions
func filterEntriesByRole(entries []models.Entry, user *models.User) []models.Entry {
	// Admins see everything
	if user.Role == models.RoleAdmin {
		return entries
	}

	// Supervisors see entries from their managed operators
	if user.Role == models.RoleSupervisor {
		if len(user.ManagedOperators) == 0 {
			return []models.Entry{}
		}

		filtered := []models.Entry{}
		for _, entry := range entries {
			for _, operatorID := range user.ManagedOperators {
				if entry.LoggingUserID == operatorID {
					filtered = append(filtered, entry)
					break
				}
			}
		}
		return filtered
	}

	// Gate operators see only their own entries
	if user.Role == models.RoleGateOperator {
		filtered := []models.Entry{}
		for _, entry := range entries {
			if entry.LoggingUserID == user.UserID {
				filtered = append(filtered, entry)
			}
		}
		return filtered
	}

	// Default: no entries
	return []models.Entry{}
}
