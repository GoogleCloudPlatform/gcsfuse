package workloadprofiler

import (
	"fmt"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"gopkg.in/yaml.v3"
	"os"
)

func yamlDumpingCallback(data map[string]interface{}, dumpDir string) {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		fmt.Printf("Error marshalling to YAML: %v\n", err)
		return
	}

	// Write the yaml data to the file.
	filePath := fmt.Sprintf("%s/profile_dump.yaml", dumpDir)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logger.Errorf("Error opening file for appending: %v\n", err)
		return
	}
	defer file.Close()

	_, err = file.Write(yamlData)
	if err != nil {
		logger.Errorf("Error writing YAML to file: %v\n", err)
		return
	}
	logger.Infof("Profile data dumped to %s", filePath)
}
