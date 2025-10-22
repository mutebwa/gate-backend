package middleware

import (
	"context"
	"encoding/json"
	"gatekeeper/auth"
	"gatekeeper/db"
	"gatekeeper/models"
	"net/http"
)

type contextKey string

const UserContextKey contextKey = "user"

// AuthMiddleware validates JWT tokens and injects user into context
func AuthMiddleware(jwtManager *auth.JWTManager, firestoreDB *db.FirestoreDB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Extract token from "Bearer <token>"
			token, err := auth.ExtractToken(authHeader)
			if err != nil {
				writeError(w, "Invalid authorization header", http.StatusUnauthorized)
				return
			}

			// Validate token
			claims, err := jwtManager.ValidateToken(token)
			if err != nil {
				writeError(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Fetch user from database to get latest data
			user, err := firestoreDB.GetUser(claims.UserID)
			if err != nil {
				writeError(w, "User not found", http.StatusUnauthorized)
				return
			}

			// Inject user into context
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserFromContext retrieves the user from the request context
func GetUserFromContext(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(UserContextKey).(*models.User)
	return user, ok
}

// RequireRole middleware checks if the user has the required role
func RequireRole(allowedRoles ...models.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUserFromContext(r.Context())
			if !ok {
				writeError(w, "User not found in context", http.StatusUnauthorized)
				return
			}

			// Check if user has one of the allowed roles
			hasRole := false
			for _, role := range allowedRoles {
				if user.Role == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				writeError(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
