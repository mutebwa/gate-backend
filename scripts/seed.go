package main

import (
	"context"
	"fmt"
	"gatekeeper/auth"
	"gatekeeper/config"
	"gatekeeper/db"
	"gatekeeper/models"
	"log"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Load configuration
	cfg := config.Load()
	cfg.Validate()

	// Initialize Firestore
	ctx := context.Background()
	firestoreDB, err := db.NewFirestoreDB(ctx, cfg.Firebase.ProjectID, cfg.Firebase.CredentialsPath)
	if err != nil {
		log.Fatalf("Failed to initialize Firestore: %v", err)
	}
	defer firestoreDB.Close()

	log.Println("🌱 Starting database seeding...")

	// Seed checkpoints
	if err := seedCheckpoints(firestoreDB); err != nil {
		log.Fatalf("Failed to seed checkpoints: %v", err)
	}

	// Seed users
	if err := seedUsers(firestoreDB); err != nil {
		log.Fatalf("Failed to seed users: %v", err)
	}

	log.Println("✅ Database seeding completed successfully!")
}

func seedCheckpoints(db *db.FirestoreDB) error {
	checkpoints := []models.Checkpoint{
		{
			CheckpointID: "CP-EAST-MAIN",
			Name:         "East Main Gate",
			Location:     "Sector 1",
		},
		{
			CheckpointID: "CP-WEST-GATE",
			Name:         "West Gate",
			Location:     "Sector 4",
		},
		{
			CheckpointID: "CP-NORTH-01",
			Name:         "North Checkpoint 1",
			Location:     "Sector 2",
		},
		{
			CheckpointID: "CP-SOUTH-01",
			Name:         "South Checkpoint 1",
			Location:     "Sector 3",
		},
	}

	for _, checkpoint := range checkpoints {
		if err := db.CreateCheckpoint(&checkpoint); err != nil {
			return fmt.Errorf("failed to create checkpoint %s: %w", checkpoint.CheckpointID, err)
		}
		log.Printf("  ✓ Created checkpoint: %s", checkpoint.Name)
	}

	return nil
}

func seedUsers(firestoreDB *db.FirestoreDB) error {
	users := []struct {
		User     models.User
		Password string
	}{
		{
			User: models.User{
				UserID:             "user-admin",
				Username:           "admin",
				Role:               models.RoleAdmin,
				AllowedCheckpoints: []string{},
				LastLogin:          time.Now(),
			},
			Password: "password",
		},
		{
			User: models.User{
				UserID:             "user-supervisor-john",
				Username:           "supervisor_john",
				Role:               models.RoleSupervisor,
				AllowedCheckpoints: []string{"CP-EAST-MAIN", "CP-NORTH-01"},
				ManagedOperators:   []string{},
				LastLogin:          time.Now(),
			},
			Password: "password",
		},
		{
			User: models.User{
				UserID:             "user-op-east",
				Username:           "op_east",
				Role:               models.RoleGateOperator,
				AllowedCheckpoints: []string{"CP-EAST-MAIN"},
				SupervisorID:       "user-supervisor-john",
				LastLogin:          time.Now(),
			},
			Password: "password",
		},
		{
			User: models.User{
				UserID:             "user-op-west",
				Username:           "op_west",
				Role:               models.RoleGateOperator,
				AllowedCheckpoints: []string{"CP-WEST-GATE"},
				LastLogin:          time.Now(),
			},
			Password: "password",
		},
	}

	for _, userData := range users {
		// Create user
		if err := firestoreDB.CreateUser(&userData.User); err != nil {
			return fmt.Errorf("failed to create user %s: %w", userData.User.Username, err)
		}

		// Hash and store password
		passwordHash, err := auth.HashPassword(userData.Password)
		if err != nil {
			return fmt.Errorf("failed to hash password for %s: %w", userData.User.Username, err)
		}

		if err := firestoreDB.StorePasswordHash(userData.User.UserID, passwordHash); err != nil {
			return fmt.Errorf("failed to store password for %s: %w", userData.User.Username, err)
		}

		log.Printf("  ✓ Created user: %s (role: %s)", userData.User.Username, userData.User.Role)
	}

	// Update supervisor's managed operators
	supervisor, err := firestoreDB.GetUser("user-supervisor-john")
	if err != nil {
		return fmt.Errorf("failed to get supervisor: %w", err)
	}

	supervisor.ManagedOperators = []string{"user-op-east"}
	if err := firestoreDB.UpdateUser(supervisor); err != nil {
		return fmt.Errorf("failed to update supervisor: %w", err)
	}

	log.Println("  ✓ Updated supervisor relationships")

	return nil
}
