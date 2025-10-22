// models.go
// Defines the core data structures used by the Go backend (Cloud API and Tauri Sidecar).

package models

import (
	"time"
)

// EntryType defines the different categories of checkpoint entries.
type EntryType string

const (
	EntryTypePersonnel EntryType = "PERSONNEL"
	EntryTypeTruck     EntryType = "TRUCK"
	EntryTypeCar       EntryType = "CAR"
	EntryTypeOther     EntryType = "OTHER"
)

// EntryStatus defines the synchronization status of a document.
type EntryStatus string

const (
	StatusActive  EntryStatus = "ACTIVE"
	StatusDeleted EntryStatus = "DELETED"
)

// Entry is the unified struct for all checkpoint entries (Personnel, Vehicle, Other).
// This struct maps directly to a Firestore document and is used for Go API request/response payloads.
type Entry struct {
	// === Core Synchronization Fields (Mandatory - See Decision 1.2) ===
	RecordID      string      `firestore:"record_id" json:"record_id"`         // Client-generated UUID (Local ID)
	CheckpointID  string      `firestore:"checkpoint_id" json:"checkpoint_id"` // FR1.3 - Checkpoint where entry occurred
	EntryType     EntryType   `firestore:"entry_type" json:"entry_type"`       // e.g., "PERSONNEL", "TRUCK"
	LoggingUserID string      `firestore:"logging_user_id" json:"logging_user_id"` // FR1.2 - User who made the entry
	ClientTS      time.Time   `firestore:"client_ts" json:"client_ts"`           // Client timestamp of submission

	// === Server-Controlled Sync Fields (Set by Go API) ===
	UpdatedAt     time.Time   `firestore:"updated_at" json:"updated_at"`         // CRITICAL: Server-authoritative timestamp for Last Write Wins
	CreatedAt     time.Time   `firestore:"created_at" json:"created_at"`         // Server-validated creation time
	Status        EntryStatus `firestore:"status" json:"status"`               // e.g., "ACTIVE", "DELETED"

	// === Type-Specific Data (Flexible Payload) ===
	// This map holds the specific data fields for the entry type.
	Payload       map[string]interface{} `firestore:"payload" json:"payload"` 
}

// AuditLog represents an audit log entry.
type AuditLog struct {
	LogID    string `firestore:"log_id" json:"log_id"`
	Timestamp string `firestore:"timestamp" json:"timestamp"`
	UserID   string `firestore:"user_id" json:"user_id"`
	Action   string `firestore:"action" json:"action"`
	Details  string `firestore:"details" json:"details"`
}

// Checkpoint represents a checkpoint in the system.
type Checkpoint struct {
	CheckpointID string `firestore:"checkpoint_id" json:"checkpoint_id"`
	Name        string `firestore:"name" json:"name"`
	Location    string `firestore:"location" json:"location"`
}

// UserRole defines the access level of a user.
type UserRole string

const (
	RoleAdmin       UserRole = "ADMIN"
	RoleSupervisor  UserRole = "SUPERVISOR"
	RoleGateOperator UserRole = "GATE_OPERATOR"
)

// User represents an authenticated user in the system.
// This struct is essential for Role-Based Access Control (RBAC).
type User struct {
	UserID             string   `firestore:"user_id" json:"user_id"`
	Username           string   `firestore:"username" json:"username"`
	Role               UserRole `firestore:"role" json:"role"` // ADMIN, SUPERVISOR, GATE_OPERATOR
	AllowedCheckpoints []string `firestore:"allowed_checkpoints" json:"allowed_checkpoints"` // Decided in Structural Decision 4.1
	SupervisorID       string   `firestore:"supervisor_id,omitempty" json:"supervisor_id,omitempty"` // For GATE_OPERATOR: which supervisor manages them
	ManagedOperators   []string `firestore:"managed_operators,omitempty" json:"managed_operators,omitempty"` // For SUPERVISOR: list of operator user_ids they manage
	LastLogin          time.Time `firestore:"last_login" json:"last_login"`
}

// AuthRequest is the payload for mock login
type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse returns the token and user details
type AuthResponse struct {
	Token string `json:"token"` // Mock Token (UserID)
	User  User   `json:"user"`
}

// PullRequest is used for the PULL operation
type PullRequest struct {
	LastSuccessfulSync time.Time `json:"last_successful_sync"`
}

// SyncRequest represents the payload sent by the client to the Go API during PUSH sync.
type SyncRequest struct {
	LastSuccessfulSync time.Time `json:"last_successful_sync"` // Used for the PULL response (Delta Sync)
	PendingEntries     []Entry   `json:"pending_entries"`      // Array of entries created/modified while offline
}

// SyncResponse represents the data returned by the Go API during PUSH/PULL sync.
type SyncResponse struct {
	Success          bool     `json:"success"`
	NewLastSyncTime  time.Time `json:"new_last_sync_time"`
	UpdatedEntries   []Entry  `json:"updated_entries"`    // Entries updated by the server or newer than client's copy
	RejectedEntryIDs []string `json:"rejected_entry_ids"` // IDs of entries rejected due to conflict (client is older)
	Error            string   `json:"error,omitempty"`
}
