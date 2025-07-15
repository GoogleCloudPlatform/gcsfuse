package workloadprofiler

import (
	"fmt"
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
	err = os.WriteFile(filePath, yamlData, 0644)
	if err != nil {
		fmt.Printf("Error writing YAML to file: %v\n", err)
		return
	}

	fmt.Print(string(yamlData))
}
