package config

type WriteConfig struct {
	// Check in unit tests if nested values return nil in anycase.
	CreateEmptyFile bool `yaml:"create-empty-file"`
}

type MountConfig struct {
	WriteConfig `yaml:"write"`
}
