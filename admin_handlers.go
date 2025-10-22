// admin_handlers.go

package main

import (
	"encoding/json"
	"fmt"
	"gatekeeper/models"
	"net/http"
)

// handleAdminCreateUser handles the creation of a new user by an admin.
func handleAdminCreateUser(w http.ResponseWriter, r *http.Request, adminUser models.User) {
	var newUser models.User
	if err := json.NewDecoder(r.Body).Decode(&newUser); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// In a real application, you would add validation here.
	// For the mock, we'll just add the user to the store.
	mockUserStore[newUser.UserID] = newUser

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newUser)
	logAuditEvent(adminUser.UserID, "ADMIN_CREATE_USER", fmt.Sprintf("Admin '%s' created new user '%s' with role '%s'", adminUser.Username, newUser.Username, newUser.Role))
}

// handleAdminUpdateUserRole handles updating a user's role.
func handleAdminUpdateUserRole(w http.ResponseWriter, r *http.Request, adminUser models.User) {
	var req struct {
		UserID string          `json:"user_id"`
		NewRole  models.UserRole `json:"new_role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	user, found := mockUserStore[req.UserID]
	if !found {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	user.Role = req.NewRole
	mockUserStore[req.UserID] = user

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
	logAuditEvent(adminUser.UserID, "ADMIN_UPDATE_ROLE", fmt.Sprintf("Admin '%s' changed role of user '%s' to '%s'", adminUser.Username, user.Username, user.Role))
}

// handleAdminCreateCheckpoint handles creating a new checkpoint.
func handleAdminCreateCheckpoint(w http.ResponseWriter, r *http.Request, adminUser models.User) {
	var newCheckpoint models.Checkpoint
	if err := json.NewDecoder(r.Body).Decode(&newCheckpoint); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	mockCheckpointStore[newCheckpoint.CheckpointID] = newCheckpoint

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newCheckpoint)
	logAuditEvent(adminUser.UserID, "ADMIN_CREATE_CHECKPOINT", fmt.Sprintf("Admin '%s' created new checkpoint '%s'", adminUser.Username, newCheckpoint.Name))
}

// handleAdminGetCheckpoints returns all checkpoints.
func handleAdminGetCheckpoints(w http.ResponseWriter, r *http.Request, adminUser models.User) {
	var checkpoints []models.Checkpoint
	for _, cp := range mockCheckpointStore {
		checkpoints = append(checkpoints, cp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(checkpoints)
}
