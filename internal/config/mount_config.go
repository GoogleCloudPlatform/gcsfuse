// Copyright 2021 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

type WriteConfig struct {
	CreateEmptyFile bool `yaml:"create-empty-file"`
}

type LogConfig struct {
	Severity  LogSeverity `yaml:"severity"`
	Format    string      `yaml:"format"`
	FilePath  string      `yaml:"file-path"`
	LogRotate LogRotate   `yaml:"log-rotate"`
}

type MountConfig struct {
	WriteConfig `yaml:"write"`
	LogConfig   `yaml:"logging"`
}

type LogRotate struct {
	MaxSizeInMB uint32 `yaml:"max-size-in-mb"`
	MaxDays     uint32 `yaml:"max-days"`
	BackupCount uint32 `yaml:"backup-count"`
	Compress    bool   `yaml:"compress"`
}

func DefaultLogRotateConfig() LogRotate {
	return LogRotate{
		MaxSizeInMB: 200,
		MaxDays:     28,
		BackupCount: 3,
		Compress:    true,
	}
}

func NewMountConfig() *MountConfig {
	mountConfig := &MountConfig{}
	mountConfig.LogConfig = LogConfig{
		// Making the default severity as INFO.
		Severity: INFO,
		// Setting default values of log rotate config.
		LogRotate: DefaultLogRotateConfig(),
	}
	return mountConfig
}
