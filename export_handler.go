// export_handler.go

package main

import (
	"encoding/csv"
	"fmt"
	"gatekeeper/models"
	"net/http"
	"os"
)

// handleExportToCSV handles the data export request.
func handleExportToCSV(w http.ResponseWriter, r *http.Request, user models.User) {
	// In a real app, you would check for Supervisor or Admin role here.

	// 1. Generate CSV content
	filePath, err := generateCSV()
	if err != nil {
		http.Error(w, "Failed to generate CSV", http.StatusInternalServerError)
		return
	}

	// 2. Mock external storage upload
	downloadURL := mockUploadToStorage(filePath)

	// 3. Return download link
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"download_url": "%s"}`, downloadURL)

	logAuditEvent(user.UserID, "DATA_EXPORT", fmt.Sprintf("User '%s' exported data", user.Username))
}

func generateCSV() (string, error) {
	filePath := "/tmp/export.csv"
	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"RecordID", "CheckpointID", "EntryType", "ClientTS", "LoggingUserID", "Status"}
	writer.Write(header)

	// Write data
	for _, entry := range mockDB {
		row := []string{entry.RecordID, entry.CheckpointID, string(entry.EntryType), entry.ClientTS.String(), entry.LoggingUserID, string(entry.Status)}
		writer.Write(row)
	}

	return filePath, nil
}

func mockUploadToStorage(filePath string) string {
	// This is a mock. In a real app, you would use an SDK for S3, GCS, etc.
	fmt.Printf("Mock Upload: Uploading %s to cloud storage...\n", filePath)
	return fmt.Sprintf("https://storage.example.com/%s", filePath)
}
