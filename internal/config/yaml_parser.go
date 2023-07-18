package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

func ReadConfigFile(fileName string) (mountConfig *MountConfig, err error) {
	buf, err := os.ReadFile(fileName)

	if err != nil {
		return
	}

	mountConfig = &MountConfig{}
	err = yaml.Unmarshal(buf, mountConfig)

	if err != nil {
		return
	}

	return
}
