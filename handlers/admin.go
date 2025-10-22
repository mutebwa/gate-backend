package handlers

import (
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

type AdminHandler struct {
	db *db.FirestoreDB
}

func NewAdminHandler(firestoreDB *db.FirestoreDB) *AdminHandler {
	return &AdminHandler{
		db: firestoreDB,
	}
}

// --- User Management ---

type CreateUserRequest struct {
	Username           string          `json:"username"`
	Password           string          `json:"password"`
	Role               models.UserRole `json:"role"`
	AllowedCheckpoints []string        `json:"allowed_checkpoints"`
	SupervisorID       string          `json:"supervisor_id,omitempty"`
}

type UpdateUserRequest struct {
	UserID             string          `json:"user_id"`
	Role               models.UserRole `json:"role,omitempty"`
	AllowedCheckpoints []string        `json:"allowed_checkpoints,omitempty"`
	SupervisorID       string          `json:"supervisor_id,omitempty"`
}

type DeleteUserRequest struct {
	UserID string `json:"user_id"`
}

// GetUsers returns all users
func (h *AdminHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	users, err := h.db.GetAllUsers()
	if err != nil {
		log.Printf("❌ Failed to get users: %v", err)
		writeError(w, "Failed to retrieve users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// CreateUser creates a new user
func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	adminUser, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		writeError(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		writeError(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Validate password strength
	if err := auth.ValidatePasswordStrength(req.Password); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if username already exists
	existingUser, _ := h.db.GetUserByUsername(req.Username)
	if existingUser != nil {
		writeError(w, "Username already exists", http.StatusConflict)
		return
	}

	// Generate user ID
	userID := fmt.Sprintf("user-%s", req.Username)

	// Create user
	user := &models.User{
		UserID:             userID,
		Username:           req.Username,
		Role:               req.Role,
		AllowedCheckpoints: req.AllowedCheckpoints,
		SupervisorID:       req.SupervisorID,
		LastLogin:          time.Now(),
	}

	if err := h.db.CreateUser(user); err != nil {
		log.Printf("❌ Failed to create user: %v", err)
		writeError(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Hash and store password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("❌ Failed to hash password: %v", err)
		writeError(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	if err := h.db.StorePasswordHash(userID, passwordHash); err != nil {
		log.Printf("❌ Failed to store password: %v", err)
		writeError(w, "Failed to store password", http.StatusInternalServerError)
		return
	}

	// If this is a gate operator with a supervisor, update the supervisor's managed operators
	if req.Role == models.RoleGateOperator && req.SupervisorID != "" {
		supervisor, err := h.db.GetUser(req.SupervisorID)
		if err == nil {
			if supervisor.ManagedOperators == nil {
				supervisor.ManagedOperators = []string{}
			}
			// Add operator to supervisor's list if not already there
			found := false
			for _, opID := range supervisor.ManagedOperators {
				if opID == userID {
					found = true
					break
				}
			}
			if !found {
				supervisor.ManagedOperators = append(supervisor.ManagedOperators, userID)
				h.db.UpdateUser(supervisor)
			}
		}
	}

	log.Printf("✅ User created by %s: %s (role: %s)", adminUser.Username, req.Username, req.Role)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// UpdateUser updates an existing user
func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	adminUser, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		writeError(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		writeError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Get existing user
	user, err := h.db.GetUser(req.UserID)
	if err != nil {
		writeError(w, "User not found", http.StatusNotFound)
		return
	}

	// Store old supervisor ID for cleanup
	oldSupervisorID := user.SupervisorID

	// Update fields
	if req.Role != "" {
		user.Role = req.Role
	}
	if req.AllowedCheckpoints != nil {
		user.AllowedCheckpoints = req.AllowedCheckpoints
	}
	if req.SupervisorID != "" {
		user.SupervisorID = req.SupervisorID
	}

	// Update user
	if err := h.db.UpdateUser(user); err != nil {
		log.Printf("❌ Failed to update user: %v", err)
		writeError(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	// Update supervisor relationships if supervisor changed
	if oldSupervisorID != req.SupervisorID {
		// Remove from old supervisor's list
		if oldSupervisorID != "" {
			oldSupervisor, err := h.db.GetUser(oldSupervisorID)
			if err == nil {
				newList := []string{}
				for _, opID := range oldSupervisor.ManagedOperators {
					if opID != req.UserID {
						newList = append(newList, opID)
					}
				}
				oldSupervisor.ManagedOperators = newList
				h.db.UpdateUser(oldSupervisor)
			}
		}

		// Add to new supervisor's list
		if req.SupervisorID != "" {
			newSupervisor, err := h.db.GetUser(req.SupervisorID)
			if err == nil {
				if newSupervisor.ManagedOperators == nil {
					newSupervisor.ManagedOperators = []string{}
				}
				found := false
				for _, opID := range newSupervisor.ManagedOperators {
					if opID == req.UserID {
						found = true
						break
					}
				}
				if !found {
					newSupervisor.ManagedOperators = append(newSupervisor.ManagedOperators, req.UserID)
					h.db.UpdateUser(newSupervisor)
				}
			}
		}
	}

	log.Printf("✅ User updated by %s: %s", adminUser.Username, user.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// DeleteUser deletes a user
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	adminUser, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		writeError(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	var req DeleteUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		writeError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Prevent deleting yourself
	if req.UserID == adminUser.UserID {
		writeError(w, "Cannot delete your own account", http.StatusBadRequest)
		return
	}

	// Get user to check supervisor relationships
	user, err := h.db.GetUser(req.UserID)
	if err != nil {
		writeError(w, "User not found", http.StatusNotFound)
		return
	}

	// Remove from supervisor's managed operators list
	if user.SupervisorID != "" {
		supervisor, err := h.db.GetUser(user.SupervisorID)
		if err == nil {
			newList := []string{}
			for _, opID := range supervisor.ManagedOperators {
				if opID != req.UserID {
					newList = append(newList, opID)
				}
			}
			supervisor.ManagedOperators = newList
			h.db.UpdateUser(supervisor)
		}
	}

	// Delete user
	if err := h.db.DeleteUser(req.UserID); err != nil {
		log.Printf("❌ Failed to delete user: %v", err)
		writeError(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ User deleted by %s: %s", adminUser.Username, user.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User deleted successfully",
	})
}

// --- Checkpoint Management ---

type CreateCheckpointRequest struct {
	CheckpointID string `json:"checkpoint_id"`
	Name         string `json:"name"`
	Location     string `json:"location"`
}

// GetCheckpoints returns all checkpoints
func (h *AdminHandler) GetCheckpoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	checkpoints, err := h.db.GetAllCheckpoints()
	if err != nil {
		log.Printf("❌ Failed to get checkpoints: %v", err)
		writeError(w, "Failed to retrieve checkpoints", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(checkpoints)
}

// CreateCheckpoint creates a new checkpoint
func (h *AdminHandler) CreateCheckpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	adminUser, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		writeError(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	var req CreateCheckpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CheckpointID == "" || req.Name == "" {
		writeError(w, "Checkpoint ID and name are required", http.StatusBadRequest)
		return
	}

	checkpoint := &models.Checkpoint{
		CheckpointID: req.CheckpointID,
		Name:         req.Name,
		Location:     req.Location,
	}

	if err := h.db.CreateCheckpoint(checkpoint); err != nil {
		log.Printf("❌ Failed to create checkpoint: %v", err)
		writeError(w, "Failed to create checkpoint", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Checkpoint created by %s: %s", adminUser.Username, req.Name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(checkpoint)
}
