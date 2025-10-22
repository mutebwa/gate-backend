package handlers

import (
	"encoding/json"
	"gatekeeper/auth"
	"gatekeeper/db"
	"gatekeeper/models"
	"log"
	"net/http"
	"time"
)

type AuthHandler struct {
	db         *db.FirestoreDB
	jwtManager *auth.JWTManager
}

func NewAuthHandler(firestoreDB *db.FirestoreDB, jwtManager *auth.JWTManager) *AuthHandler {
	return &AuthHandler{
		db:         firestoreDB,
		jwtManager: jwtManager,
	}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token        string       `json:"token"`
	RefreshToken string       `json:"refresh_token"`
	User         *models.User `json:"user"`
}

// Login handles user authentication
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		writeError(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Get user by username
	user, err := h.db.GetUserByUsername(req.Username)
	if err != nil {
		log.Printf("Login failed for user %s: user not found", req.Username)
		writeError(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Get password hash
	passwordHash, err := h.db.GetPasswordHash(user.UserID)
	if err != nil {
		log.Printf("Login failed for user %s: password hash not found", req.Username)
		writeError(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Verify password
	if err := auth.CheckPassword(req.Password, passwordHash); err != nil {
		log.Printf("Login failed for user %s: invalid password", req.Username)
		writeError(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Update last login
	user.LastLogin = time.Now()
	if err := h.db.UpdateUser(user); err != nil {
		log.Printf("Warning: failed to update last login for user %s: %v", req.Username, err)
	}

	// Generate tokens
	token, err := h.jwtManager.GenerateToken(user)
	if err != nil {
		log.Printf("Failed to generate token for user %s: %v", req.Username, err)
		writeError(w, "Failed to generate authentication token", http.StatusInternalServerError)
		return
	}

	refreshToken, err := h.jwtManager.GenerateRefreshToken(user)
	if err != nil {
		log.Printf("Failed to generate refresh token for user %s: %v", req.Username, err)
		writeError(w, "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ User logged in: %s (role: %s)", user.Username, user.Role)

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         user,
	})
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type RefreshTokenResponse struct {
	Token string `json:"token"`
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate refresh token
	claims, err := h.jwtManager.ValidateToken(req.RefreshToken)
	if err != nil {
		writeError(w, "Invalid or expired refresh token", http.StatusUnauthorized)
		return
	}

	// Get user
	user, err := h.db.GetUser(claims.UserID)
	if err != nil {
		writeError(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Generate new access token
	token, err := h.jwtManager.GenerateToken(user)
	if err != nil {
		log.Printf("Failed to generate token for user %s: %v", user.Username, err)
		writeError(w, "Failed to generate authentication token", http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(RefreshTokenResponse{
		Token: token,
	})
}

func writeError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
