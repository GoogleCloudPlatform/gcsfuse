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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeProtoFields(t *testing.T) {
	params := []Param{
		{
			FlagName:       "file-cache-max-size-mb",
			ConfigPath:     "file-cache.max-size-mb",
			ProtoType:      "int32",
			ProtoFieldName: "file_cache_max_size_mb",
			ProtoTag:       3,
		},
		{
			FlagName:       "app-name",
			ConfigPath:     "app-name",
			ProtoType:      "bool",
			ProtoFieldName: "is_app_name_set",
			ProtoTag:       1,
		},
		{
			FlagName:       "cache-dir",
			ConfigPath:     "cache-dir",
			ProtoType:      "bool",
			ProtoFieldName: "is_cache_dir_set",
			ProtoTag:       2,
		},
		{
			FlagName:       "deprecated-cli-flag",
			ConfigPath:     "",
			ProtoType:      "bool",
			ProtoFieldName: "is_deprecated_cli_flag_set",
			ProtoTag:       0,
		},
	}

	protoFields := computeProtoFields(params)

	require.Len(t, protoFields, 3)
	assert.Equal(t, "is_app_name_set", protoFields[0].ProtoFieldName)
	assert.Equal(t, 1, protoFields[0].ProtoTag)
	assert.Equal(t, "bool", protoFields[0].ProtoType)

	assert.Equal(t, "is_cache_dir_set", protoFields[1].ProtoFieldName)
	assert.Equal(t, 2, protoFields[1].ProtoTag)
	assert.Equal(t, "bool", protoFields[1].ProtoType)

	assert.Equal(t, "file_cache_max_size_mb", protoFields[2].ProtoFieldName)
	assert.Equal(t, 3, protoFields[2].ProtoTag)
	assert.Equal(t, "int32", protoFields[2].ProtoType)
}

func TestFormatReservedTags(t *testing.T) {
	assert.Equal(t, "", formatReservedTags(nil))
	assert.Equal(t, "", formatReservedTags([]int{}))
	assert.Equal(t, "4", formatReservedTags([]int{4}))
	assert.Equal(t, "4, 10, 28", formatReservedTags([]int{28, 4, 10}))
}

func TestConstructTypeTemplateData(t *testing.T) {
	params := []Param{
		{
			FlagName:   "app-name",
			ConfigPath: "app-name",
			Type:       "string",
		},
		{
			FlagName:   "file-cache-max-size-mb",
			ConfigPath: "file-cache.max-size-mb",
			Type:       "int",
		},
		{
			FlagName:   "logging-severity",
			ConfigPath: "logging.severity",
			Type:       "logSeverity",
		},
	}

	ttd, err := constructTypeTemplateData(params)
	require.NoError(t, err)

	var configMsg, fileCacheMsg, loggingMsg typeTemplateData
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

	assert.Equal(t, "Config", configMsg.TypeName)
	require.Len(t, configMsg.Fields, 3)
	assert.Equal(t, "AppName", configMsg.Fields[0].FieldName)
	assert.Equal(t, "FileCache", configMsg.Fields[1].FieldName)
	assert.Equal(t, "Logging", configMsg.Fields[2].FieldName)

	assert.Equal(t, "FileCacheConfig", fileCacheMsg.TypeName)
	require.Len(t, fileCacheMsg.Fields, 1)
	assert.Equal(t, "MaxSizeMb", fileCacheMsg.Fields[0].FieldName)

	assert.Equal(t, "LoggingConfig", loggingMsg.TypeName)
	require.Len(t, loggingMsg.Fields, 1)
	assert.Equal(t, "Severity", loggingMsg.Fields[0].FieldName)
}
