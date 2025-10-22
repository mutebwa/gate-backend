package main

import (
	"gatekeeper/models"
	"sync"
	"time"
)

// --- Mock Database (In-Memory for demonstration only) ---

var mockDB = make(map[string]models.Entry)
var dbMutex sync.RWMutex

// mockUserStore simulates the User collection lookup
var mockUserStore = map[string]models.User{
	"user-op-east": {
		UserID:             "user-op-east",
		Username:           "op_east",
		Role:               models.RoleGateOperator,
		AllowedCheckpoints: []string{"CP-EAST-MAIN"},
	},
	"user-op-west": {
		UserID:             "user-op-west",
		Username:           "op_west",
		Role:               models.RoleGateOperator,
		AllowedCheckpoints: []string{"CP-WEST-GATE"},
	},
	"user-admin": {
		UserID:             "user-admin",
		Username:           "admin",
		Role:               models.RoleAdmin,
		AllowedCheckpoints: []string{}, // Admins can see all
	},
}

var mockUserCreds = map[string]string{
	"op_east": "password",
	"op_west": "password",
	"admin":   "password", // Fixed: Now matches frontend expectation
}

var mockCheckpointStore = map[string]models.Checkpoint{
	"CP-EAST-MAIN": {CheckpointID: "CP-EAST-MAIN", Name: "East Main Gate", Location: "Sector 1"},
	"CP-WEST-GATE": {CheckpointID: "CP-WEST-GATE", Name: "West Gate", Location: "Sector 4"},
}

// --- Mock DB Access Functions ---

func mockGetEntry(recordID string) (models.Entry, bool) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()
	entry, found := mockDB[recordID]
	return entry, found
}

func mockSaveEntry(entry models.Entry) {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	mockDB[entry.RecordID] = entry
}

// mockQueryUpdatedSince filters records based on time and the user's allowed checkpoints (FR1.3 enforcement)
func mockQueryUpdatedSince(lastSync time.Time, allowedCheckpoints []string) []models.Entry {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	var results []models.Entry
	isGlobalAdmin := len(allowedCheckpoints) == 0 // Admin/Supervisor have empty list for global access

	for _, entry := range mockDB {
		// 1. Filter by time
		if entry.UpdatedAt.After(lastSync) {

			// 2. Filter by Checkpoint access (FR1.3 enforcement)
			if isGlobalAdmin {
				results = append(results, entry)
				continue
			}

			// Check if the entry's checkpoint is in the operator's allowed list
			for _, cpID := range allowedCheckpoints {
				if entry.CheckpointID == cpID {
					results = append(results, entry)
					break
				}
			}
		}
	}
	return results
}
