package setup

import (
	"fmt"
	"os"
	"path"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
	"gopkg.in/yaml.v3"
)

func YAMLConfigFile(config config.MountConfig) (filePath string) {
	yamlData, err := yaml.Marshal(&config)
	if err != nil {
		LogAndExit(fmt.Sprintf("Error while marshaling config file: %v", err))
	}

	fileName := "config.yaml"
	filePath = path.Join(TestDir(), fileName)
	err = os.WriteFile(filePath, yamlData, 0644)
	if err != nil {
		LogAndExit("Unable to write data into config file.")
	}
	return
}
