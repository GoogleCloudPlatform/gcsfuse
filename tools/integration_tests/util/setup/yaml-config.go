package setup

import (
	"fmt"
	"os"
	"path"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"gopkg.in/yaml.v3"
)

func YAMLConfigFile(config config.MountConfig, fileName string) (filePath string) {
	yamlData, err := yaml.Marshal(&config)
	if err != nil {
		LogAndExit(fmt.Sprintf("Error while marshaling config file: %v", err))
	}

	filePath = path.Join(TestDir(), fileName)
	err = os.WriteFile(filePath, yamlData, 0644)
	if err != nil {
		LogAndExit("Unable to write data into config file.")
	}
	return
}
