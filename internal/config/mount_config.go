package config

type WriteConfig struct {
	CreateEmptyFile bool `yaml:"create-empty-file"`
}

type MountConfig struct {
	WriteConfig `yaml:"write"`
}

func NewMountConfig() *MountConfig {
	return &MountConfig{
		WriteConfig{
			// Making the default value as true to keep it inline with current behaviour.
			CreateEmptyFile: true,
		},
	}
}
