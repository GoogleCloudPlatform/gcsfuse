package config

type WriteConfig struct {
	CreateEmptyFile bool `yaml:"create-empty-file"`
}

type MountConfig struct {
	WriteConfig `yaml:"write"`
}
