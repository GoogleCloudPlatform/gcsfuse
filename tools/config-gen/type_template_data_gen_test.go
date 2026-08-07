/*
 * Copyright 2026 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseExistingProtoTags(t *testing.T) {
	tempDir := t.TempDir()
	protoFilePath := filepath.Join(tempDir, "config.proto")

	protoContent := `
message FileCacheConfig {
  bool is_file_cache_cache_file_for_range_read_set = 1;
  int32 file_cache_max_size_mb = 2;
}

message Config {
  bool is_app_name_set = 1;
  FileCacheConfig file_cache = 2;
}
`
	err := os.WriteFile(protoFilePath, []byte(protoContent), 0644)
	require.NoError(t, err)

	tags, err := parseExistingProtoTags(protoFilePath)
	require.NoError(t, err)

	require.Contains(t, tags, "FileCacheConfig")
	assert.Equal(t, 1, tags["FileCacheConfig"]["is_file_cache_cache_file_for_range_read_set"])
	assert.Equal(t, 2, tags["FileCacheConfig"]["file_cache_max_size_mb"])

	require.Contains(t, tags, "Config")
	assert.Equal(t, 1, tags["Config"]["is_app_name_set"])
	assert.Equal(t, 2, tags["Config"]["file_cache"])
}

func TestConstructTypeTemplateData_TagAssignment(t *testing.T) {
	tempDir := t.TempDir()
	protoFilePath := filepath.Join(tempDir, "config.proto")

	protoContent := `
message FileCacheConfig {
  int32 max_size_mb = 1;
}

message Config {
  bool is_app_name_set = 1;
  FileCacheConfig file_cache = 2;
}
`
	err := os.WriteFile(protoFilePath, []byte(protoContent), 0644)
	require.NoError(t, err)

	existingTags, err := parseExistingProtoTags(protoFilePath)
	require.NoError(t, err)

	params := []Param{
		// Existing root flag
		{
			FlagName:       "app-name",
			ConfigPath:     "app-name",
			ProtoType:      "bool",
			ProtoFieldName: "is_app_name_set",
		},
		// Existing nested flag
		{
			FlagName:       "file-cache-max-size-mb",
			ConfigPath:     "file-cache.max-size-mb",
			ProtoType:      "int32",
			ProtoFieldName: "max_size_mb",
		},
		// Brand new nested flag in existing struct
		{
			FlagName:       "file-cache-cache-dir",
			ConfigPath:     "file-cache.cache-dir",
			ProtoType:      "bool",
			ProtoFieldName: "is_cache_dir_set",
		},
		// Deprecated flag (should be skipped entirely)
		{
			FlagName:       "deprecated-flag",
			ConfigPath:     "",
			ProtoType:      "bool",
			ProtoFieldName: "is_deprecated_flag_set",
		},
		// Brand new root flag
		{
			FlagName:       "foreground",
			ConfigPath:     "foreground",
			ProtoType:      "bool",
			ProtoFieldName: "foreground",
		},
		// Brand new flag in a brand new nested struct
		{
			FlagName:       "logging-severity",
			ConfigPath:     "logging.severity",
			ProtoType:      "string",
			ProtoFieldName: "severity",
		},
	}

	ttd, err := constructTypeTemplateData(params, existingTags)
	require.NoError(t, err)

	// Validate Config message
	var configMsg typeTemplateData
	var fileCacheMsg typeTemplateData
	var loggingMsg typeTemplateData
	for _, msg := range ttd {
		switch msg.TypeName {
		case "Config":
			configMsg = msg
		case "FileCacheConfig":
			fileCacheMsg = msg
		case "LoggingConfig":
			loggingMsg = msg
		}
	}

	require.Equal(t, "Config", configMsg.TypeName)
	require.Len(t, configMsg.Fields, 4) // deprecated-flag is skipped!
	// Alphabetical order by YAML flag name: app-name, file-cache, foreground, logging
	assert.Equal(t, "is_app_name_set", configMsg.Fields[0].ProtoFieldName)
	assert.Equal(t, 1, configMsg.Fields[0].ProtoTag) // Retained from existingTags
	assert.Equal(t, "file_cache", configMsg.Fields[1].ProtoFieldName)
	assert.Equal(t, 2, configMsg.Fields[1].ProtoTag) // Retained from existingTags
	assert.Equal(t, "foreground", configMsg.Fields[2].ProtoFieldName)
	assert.Equal(t, 3, configMsg.Fields[2].ProtoTag) // maxTag (2) + 1
	assert.Equal(t, "logging", configMsg.Fields[3].ProtoFieldName)
	assert.Equal(t, 4, configMsg.Fields[3].ProtoTag) // maxTag (3) + 1

	require.Equal(t, "FileCacheConfig", fileCacheMsg.TypeName)
	require.Len(t, fileCacheMsg.Fields, 2)
	// Alphabetical order by YAML flag name: file-cache-cache-dir, file-cache-max-size-mb
	assert.Equal(t, "is_cache_dir_set", fileCacheMsg.Fields[0].ProtoFieldName)
	assert.Equal(t, 2, fileCacheMsg.Fields[0].ProtoTag) // maxTag (1) + 1
	assert.Equal(t, "max_size_mb", fileCacheMsg.Fields[1].ProtoFieldName)
	assert.Equal(t, 1, fileCacheMsg.Fields[1].ProtoTag) // Retained from existingTags

	require.Equal(t, "LoggingConfig", loggingMsg.TypeName)
	require.Len(t, loggingMsg.Fields, 1)
	assert.Equal(t, "severity", loggingMsg.Fields[0].ProtoFieldName)
	assert.Equal(t, 1, loggingMsg.Fields[0].ProtoTag) // Brand new message gets tag 1
}
