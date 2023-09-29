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

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type LogSeverity string

const (
	TRACE   LogSeverity = "TRACE"
	DEBUG   LogSeverity = "DEBUG"
	INFO    LogSeverity = "INFO"
	WARNING LogSeverity = "WARNING"
	ERROR   LogSeverity = "ERROR"
	OFF     LogSeverity = "OFF"
)

func IsValidLogSeverity(severity LogSeverity) bool {
	switch severity {
	case
		TRACE,
		DEBUG,
		INFO,
		WARNING,
		ERROR,
		OFF:
		return true
	}
	return false
}

func validateFileCacheConfig(fileCacheConfig FileCacheConfig) error {
	if fileCacheConfig.TTL < -1 || fileCacheConfig.Size < -1 {
		return fmt.Errorf("the value of size or ttl for file cache can't be less than -1")
	}
	return nil
}

func validateMetadataCacheConfig(metadataCacheConfig MetadataCacheConfig) error {
	if metadataCacheConfig.TTL < -1 || metadataCacheConfig.Capacity < -1 {
		return fmt.Errorf("the value of capacity or ttl for metadata cache can't be less than -1")
	}
	return nil
}

func validateTypeCacheConfig(typeCacheConfig TypeCacheConfig) error {
	if typeCacheConfig.TTL < -1 {
		return fmt.Errorf("the value of ttl for type cache can't be less than -1")
	}
	return nil
}

func ParseConfigFile(fileName string) (mountConfig *MountConfig, err error) {
	mountConfig = NewMountConfig()

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
	// convert log severity to upper-case
	mountConfig.LogConfig.Severity = LogSeverity(strings.ToUpper(string(mountConfig.LogConfig.Severity)))
	if !IsValidLogSeverity(mountConfig.LogConfig.Severity) {
		err = fmt.Errorf("error parsing config file: log severity should be one of [trace, debug, info, warning, error, off]")
		return
	}

	if err = validateFileCacheConfig(mountConfig.FileCacheConfig); err != nil {
		return
	}

	if err = validateMetadataCacheConfig(mountConfig.MetadataCacheConfig); err != nil {
		return
	}

	if err = validateTypeCacheConfig(mountConfig.TypeCacheConfig); err != nil {
		return
	}
	return
}
