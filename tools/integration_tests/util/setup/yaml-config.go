package setup

import (
	"fmt"
	"os"
	"path"

	config2 "github.com/googlecloudplatform/gcsfuse/internal/config"
	"gopkg.in/yaml.v2"
)

func YAMLConfig(createEmptyFile bool) (filepath string) {
	config := config2.MountConfig{
		WriteConfig: config2.WriteConfig{CreateEmptyFile: createEmptyFile},
	}

	yamlData, err := yaml.Marshal(&config)
	if err != nil {
		LogAndExit(fmt.Sprintf("Error while marshaling config file: %v", err))
	}

	fileName := "config.yaml"
	filepath = path.Join(TestDir(), fileName)
	err = os.WriteFile(filepath, yamlData, 0644)
	if err != nil {
		LogAndExit("Unable to write data into config file.")
	}
	return
}
