package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func ReadConfigFile(fileName string) (mountConfig *MountConfig, err error) {
	buf, err := os.ReadFile(fileName)

	if err != nil {
		err = fmt.Errorf("error reading config file: %w", err)
		return
	}

	mountConfig = &MountConfig{}
	err = yaml.Unmarshal(buf, mountConfig)

	if err != nil {
		err = fmt.Errorf("error parsing config file: %w", err)
		return
	}

	return
}
