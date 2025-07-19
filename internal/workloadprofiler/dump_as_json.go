package workloadprofiler

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

// jsonDumpingCallback is a DumpCallback that formats the profile data as JSON
// and prints it to the console.
func jsonDumpingCallback(data map[string]interface{}, dumpDir string) {
	// MarshalIndent provides an indented, human-readable JSON output.
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		logger.Errorf("Error marshalling to JSON: %v\n", err)
		return
	}

	if len(jsonData) == 0 {
		logger.Info("No profile data to dump.")
		return
	}

	// Write the json data to the file.
	filePath := fmt.Sprintf("%s/profile_dump.json", dumpDir)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logger.Errorf("Error opening file for appending: %v\n", err)
		return
	}
	defer file.Close()
	_, err = file.Write(jsonData)
	if err != nil {
		logger.Errorf("Error writing JSON to file: %v\n", err)
		return
	}
}
