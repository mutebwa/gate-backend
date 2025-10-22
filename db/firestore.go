package db

import (
	"context"
	"fmt"
	"gatekeeper/models"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// FirestoreDB wraps the Firestore client
type FirestoreDB struct {
	client *firestore.Client
	ctx    context.Context
}

// NewFirestoreDB initializes a new Firestore client
func NewFirestoreDB(ctx context.Context, projectID, credentialsPath string) (*FirestoreDB, error) {
	opt := option.WithCredentialsFile(credentialsPath)
	
	config := &firebase.Config{ProjectID: projectID}
	app, err := firebase.NewApp(ctx, config, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing Firebase app: %w", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("error initializing Firestore client: %w", err)
	}

	log.Printf("✅ Connected to Firestore project: %s", projectID)

	return &FirestoreDB{
		client: client,
		ctx:    ctx,
	}, nil
}

// Close closes the Firestore client
func (db *FirestoreDB) Close() error {
	return db.client.Close()
}

// --- Entry Operations ---

// CreateEntry creates a new entry in Firestore
func (db *FirestoreDB) CreateEntry(entry *models.Entry) error {
	_, err := db.client.Collection("entries").Doc(entry.RecordID).Set(db.ctx, entry)
	if err != nil {
		return fmt.Errorf("failed to create entry: %w", err)
	}
	return nil
}

// GetEntry retrieves an entry by ID
func (db *FirestoreDB) GetEntry(recordID string) (*models.Entry, error) {
	doc, err := db.client.Collection("entries").Doc(recordID).Get(db.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get entry: %w", err)
	}

	var entry models.Entry
	if err := doc.DataTo(&entry); err != nil {
		return nil, fmt.Errorf("failed to parse entry: %w", err)
	}

	return &entry, nil
}

// GetAllEntries retrieves all entries
func (db *FirestoreDB) GetAllEntries() ([]models.Entry, error) {
	iter := db.client.Collection("entries").Documents(db.ctx)
	defer iter.Stop()

	var entries []models.Entry
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate entries: %w", err)
		}

		var entry models.Entry
		if err := doc.DataTo(&entry); err != nil {
			log.Printf("Warning: failed to parse entry %s: %v", doc.Ref.ID, err)
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// GetEntriesByUser retrieves entries for a specific user
func (db *FirestoreDB) GetEntriesByUser(userID string) ([]models.Entry, error) {
	iter := db.client.Collection("entries").
		Where("logging_user_id", "==", userID).
		Documents(db.ctx)
	defer iter.Stop()

	var entries []models.Entry
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate entries: %w", err)
		}

		var entry models.Entry
		if err := doc.DataTo(&entry); err != nil {
			log.Printf("Warning: failed to parse entry %s: %v", doc.Ref.ID, err)
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// GetEntriesByCheckpoint retrieves entries for a specific checkpoint
func (db *FirestoreDB) GetEntriesByCheckpoint(checkpointID string) ([]models.Entry, error) {
	iter := db.client.Collection("entries").
		Where("checkpoint_id", "==", checkpointID).
		Documents(db.ctx)
	defer iter.Stop()

	var entries []models.Entry
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate entries: %w", err)
		}

		var entry models.Entry
		if err := doc.DataTo(&entry); err != nil {
			log.Printf("Warning: failed to parse entry %s: %v", doc.Ref.ID, err)
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// GetEntriesSince retrieves entries created after a specific timestamp
func (db *FirestoreDB) GetEntriesSince(since time.Time) ([]models.Entry, error) {
	iter := db.client.Collection("entries").
		Where("created_at", ">", since).
		Documents(db.ctx)
	defer iter.Stop()

	var entries []models.Entry
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate entries: %w", err)
		}

		var entry models.Entry
		if err := doc.DataTo(&entry); err != nil {
			log.Printf("Warning: failed to parse entry %s: %v", doc.Ref.ID, err)
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// --- User Operations ---

// CreateUser creates a new user in Firestore
func (db *FirestoreDB) CreateUser(user *models.User) error {
	_, err := db.client.Collection("users").Doc(user.UserID).Set(db.ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetUser retrieves a user by ID
func (db *FirestoreDB) GetUser(userID string) (*models.User, error) {
	doc, err := db.client.Collection("users").Doc(userID).Get(db.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	var user models.User
	if err := doc.DataTo(&user); err != nil {
		return nil, fmt.Errorf("failed to parse user: %w", err)
	}

	return &user, nil
}

// GetUserByUsername retrieves a user by username
func (db *FirestoreDB) GetUserByUsername(username string) (*models.User, error) {
	iter := db.client.Collection("users").
		Where("username", "==", username).
		Limit(1).
		Documents(db.ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("user not found: %s", username)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	var user models.User
	if err := doc.DataTo(&user); err != nil {
		return nil, fmt.Errorf("failed to parse user: %w", err)
	}

	return &user, nil
}

// GetAllUsers retrieves all users
func (db *FirestoreDB) GetAllUsers() ([]models.User, error) {
	iter := db.client.Collection("users").Documents(db.ctx)
	defer iter.Stop()

	var users []models.User
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate users: %w", err)
		}

		var user models.User
		if err := doc.DataTo(&user); err != nil {
			log.Printf("Warning: failed to parse user %s: %v", doc.Ref.ID, err)
			continue
		}
		users = append(users, user)
	}

	return users, nil
}

// UpdateUser updates an existing user
func (db *FirestoreDB) UpdateUser(user *models.User) error {
	_, err := db.client.Collection("users").Doc(user.UserID).Set(db.ctx, user)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// DeleteUser deletes a user
func (db *FirestoreDB) DeleteUser(userID string) error {
	_, err := db.client.Collection("users").Doc(userID).Delete(db.ctx)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// --- Checkpoint Operations ---

// CreateCheckpoint creates a new checkpoint in Firestore
func (db *FirestoreDB) CreateCheckpoint(checkpoint *models.Checkpoint) error {
	_, err := db.client.Collection("checkpoints").Doc(checkpoint.CheckpointID).Set(db.ctx, checkpoint)
	if err != nil {
		return fmt.Errorf("failed to create checkpoint: %w", err)
	}
	return nil
}

// GetCheckpoint retrieves a checkpoint by ID
func (db *FirestoreDB) GetCheckpoint(checkpointID string) (*models.Checkpoint, error) {
	doc, err := db.client.Collection("checkpoints").Doc(checkpointID).Get(db.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get checkpoint: %w", err)
	}

	var checkpoint models.Checkpoint
	if err := doc.DataTo(&checkpoint); err != nil {
		return nil, fmt.Errorf("failed to parse checkpoint: %w", err)
	}

	return &checkpoint, nil
}

// GetAllCheckpoints retrieves all checkpoints
func (db *FirestoreDB) GetAllCheckpoints() ([]models.Checkpoint, error) {
	iter := db.client.Collection("checkpoints").Documents(db.ctx)
	defer iter.Stop()

	var checkpoints []models.Checkpoint
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate checkpoints: %w", err)
		}

		var checkpoint models.Checkpoint
		if err := doc.DataTo(&checkpoint); err != nil {
			log.Printf("Warning: failed to parse checkpoint %s: %v", doc.Ref.ID, err)
			continue
		}
		checkpoints = append(checkpoints, checkpoint)
	}

	return checkpoints, nil
}

// UpdateCheckpoint updates an existing checkpoint
func (db *FirestoreDB) UpdateCheckpoint(checkpoint *models.Checkpoint) error {
	_, err := db.client.Collection("checkpoints").Doc(checkpoint.CheckpointID).Set(db.ctx, checkpoint)
	if err != nil {
		return fmt.Errorf("failed to update checkpoint: %w", err)
	}
	return nil
}

// DeleteCheckpoint deletes a checkpoint
func (db *FirestoreDB) DeleteCheckpoint(checkpointID string) error {
	_, err := db.client.Collection("checkpoints").Doc(checkpointID).Delete(db.ctx)
	if err != nil {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}
	return nil
}

// --- Password Operations ---

// StorePasswordHash stores a password hash for a user
func (db *FirestoreDB) StorePasswordHash(userID, passwordHash string) error {
	_, err := db.client.Collection("passwords").Doc(userID).Set(db.ctx, map[string]interface{}{
		"user_id":       userID,
		"password_hash": passwordHash,
		"updated_at":    time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to store password hash: %w", err)
	}
	return nil
}

// GetPasswordHash retrieves a password hash for a user
func (db *FirestoreDB) GetPasswordHash(userID string) (string, error) {
	doc, err := db.client.Collection("passwords").Doc(userID).Get(db.ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get password hash: %w", err)
	}

	data := doc.Data()
	if hash, ok := data["password_hash"].(string); ok {
		return hash, nil
	}

	return "", fmt.Errorf("password hash not found for user: %s", userID)
}
