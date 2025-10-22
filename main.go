// main.go
// GateKeeper Central API - Production Ready
// Implements JWT authentication, Firestore database, and comprehensive middleware

package main

import (
	"context"
	"fmt"
	"gatekeeper/auth"
	"gatekeeper/config"
	"gatekeeper/db"
	"gatekeeper/handlers"
	"gatekeeper/middleware"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

// Global instances
var (
	cfg              *config.Config
	firestoreDB      *db.FirestoreDB
	jwtManager       *auth.JWTManager
	authHandler      *handlers.AuthHandler
	syncHandler      *handlers.SyncHandler
	adminHandler     *handlers.AdminHandler
	supervisorHandler *handlers.SupervisorHandler
	rateLimiter      *middleware.RateLimiter
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, using system environment variables")
	}

	// Load configuration
	cfg = config.Load()
	cfg.Validate()

	log.Printf("🚀 Starting GateKeeper API Server")
	log.Printf("📍 Environment: %s", cfg.Server.Environment)
	log.Printf("🔧 Port: %s", cfg.Server.Port)

	// Initialize Firestore
	ctx := context.Background()
	var err error
	firestoreDB, err = db.NewFirestoreDB(ctx, cfg.Firebase.ProjectID, cfg.Firebase.CredentialsPath)
	if err != nil {
		log.Fatalf("❌ Failed to initialize Firestore: %v", err)
	}
	defer firestoreDB.Close()

	// Initialize JWT Manager
	jwtManager = auth.NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.Expiration,
		cfg.JWT.RefreshTokenExpiration,
	)
	log.Printf("🔐 JWT Manager initialized (expiration: %v)", cfg.JWT.Expiration)

	// Initialize handlers
	authHandler = handlers.NewAuthHandler(firestoreDB, jwtManager)
	syncHandler = handlers.NewSyncHandler(firestoreDB)
	adminHandler = handlers.NewAdminHandler(firestoreDB)
	supervisorHandler = handlers.NewSupervisorHandler(firestoreDB)
	log.Printf("✅ Handlers initialized")

	// Initialize rate limiter
	rateLimiter = middleware.NewRateLimiter(cfg.RateLimit.Requests, cfg.RateLimit.Window)
	rateLimiter.CleanupOldLimiters()
	log.Printf("🛡️  Rate limiter initialized (%d requests per %v)", cfg.RateLimit.Requests, cfg.RateLimit.Window)

	// Set up router
	mux := http.NewServeMux()

	// Public routes (no authentication required)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/api/login", authHandler.Login)
	mux.HandleFunc("/api/refresh", authHandler.RefreshToken)

	// Protected routes (authentication required)
	authMiddleware := middleware.AuthMiddleware(jwtManager, firestoreDB)
	
	// Sync endpoints
	mux.Handle("/api/sync/push", authMiddleware(http.HandlerFunc(syncHandler.Push)))
	mux.Handle("/api/sync/pull", authMiddleware(http.HandlerFunc(syncHandler.Pull)))

	// Admin endpoints (admin only)
	adminOnly := middleware.RequireRole("ADMIN")
	mux.Handle("/api/admin/users", authMiddleware(adminOnly(http.HandlerFunc(adminHandler.GetUsers))))
	mux.Handle("/api/admin/users/create", authMiddleware(adminOnly(http.HandlerFunc(adminHandler.CreateUser))))
	mux.Handle("/api/admin/users/update", authMiddleware(adminOnly(http.HandlerFunc(adminHandler.UpdateUser))))
	mux.Handle("/api/admin/users/delete", authMiddleware(adminOnly(http.HandlerFunc(adminHandler.DeleteUser))))
	mux.Handle("/api/admin/checkpoints", authMiddleware(adminOnly(http.HandlerFunc(adminHandler.GetCheckpoints))))
	mux.Handle("/api/admin/checkpoints/create", authMiddleware(adminOnly(http.HandlerFunc(adminHandler.CreateCheckpoint))))

	// Supervisor endpoints (supervisor or admin)
	supervisorOrAdmin := middleware.RequireRole("SUPERVISOR", "ADMIN")
	mux.Handle("/api/supervisor/entries", authMiddleware(supervisorOrAdmin(http.HandlerFunc(supervisorHandler.GetEntries))))
	mux.Handle("/api/supervisor/export", authMiddleware(supervisorOrAdmin(http.HandlerFunc(supervisorHandler.ExportEntries))))
	mux.Handle("/api/supervisor/reset-password", authMiddleware(supervisorOrAdmin(http.HandlerFunc(supervisorHandler.ResetPassword))))

	// Apply global middleware
	handler := middleware.CORSMiddleware(cfg.CORS.AllowedOrigins)(mux)
	handler = rateLimiter.Middleware()(handler)

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("✅ Server listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("❌ Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server stopped gracefully")
}

// Health check endpoint
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","timestamp":%d,"version":"1.0.0"}`, time.Now().Unix())
}
