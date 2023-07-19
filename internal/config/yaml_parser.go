package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func ParseConfigFile(fileName string) (mountConfig *MountConfig, err error) {
	mountConfig = &MountConfig{}

	if fileName == "" {
		return
	}

	buf, err := os.ReadFile(fileName)
	if err != nil {
		err = fmt.Errorf("error reading config file: %w", err)
		return
	}

	err = yaml.Unmarshal(buf, mountConfig)
	if err != nil {
		err = fmt.Errorf("error parsing config file: %w", err)
		return
	}

	return
}
