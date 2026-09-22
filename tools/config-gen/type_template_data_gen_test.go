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
	testCases := []struct {
		param             Param
		expectedFieldName string
		expectedTag       int
		expectedType      string
		isSkipped         bool
	}{
		{
			param: Param{
				FlagName:       "app-name",
				ConfigPath:     "app-name",
				ProtoType:      "bool",
				ProtoFieldName: "is_app_name_set",
				ProtoTag:       1,
			},
			expectedFieldName: "is_app_name_set",
			expectedTag:       1,
			expectedType:      "bool",
		},
		{
			param: Param{
				FlagName:       "cache-dir",
				ConfigPath:     "cache-dir",
				ProtoType:      "bool",
				ProtoFieldName: "is_cache_dir_set",
				ProtoTag:       2,
			},
			expectedFieldName: "is_cache_dir_set",
			expectedTag:       2,
			expectedType:      "bool",
		},
		{
			param: Param{
				FlagName:       "file-cache-max-size-mb",
				ConfigPath:     "file-cache.max-size-mb",
				ProtoType:      "sint64",
				ProtoFieldName: "file_cache_max_size_mb",
				ProtoTag:       3,
			},
			expectedFieldName: "file_cache_max_size_mb",
			expectedTag:       3,
			expectedType:      "sint64",
		},
		{
			param: Param{
				FlagName:       "deprecated-cli-flag",
				ConfigPath:     "",
				ProtoType:      "bool",
				ProtoFieldName: "is_deprecated_cli_flag_set",
				ProtoTag:       0,
			},
			isSkipped: true,
		},
	}
	params := []Param{
		testCases[2].param,
		testCases[0].param,
		testCases[1].param,
		testCases[3].param,
	}

	protoFields := computeProtoFields(params)

	require.Len(t, protoFields, 3)
	fieldIdx := 0
	for _, tc := range testCases {
		if tc.isSkipped {
			continue
		}
		assert.Equal(t, tc.expectedFieldName, protoFields[fieldIdx].ProtoFieldName)
		assert.Equal(t, tc.expectedTag, protoFields[fieldIdx].ProtoTag)
		assert.Equal(t, tc.expectedType, protoFields[fieldIdx].ProtoType)
		fieldIdx++
	}
}

func TestFormatReservedTags(t *testing.T) {
	testCases := []struct {
		name     string
		tags     []int
		expected string
	}{
		{
			name:     "NilSlice",
			tags:     nil,
			expected: "",
		},
		{
			name:     "EmptySlice",
			tags:     []int{},
			expected: "",
		},
		{
			name:     "SingleTag",
			tags:     []int{4},
			expected: "4",
		},
		{
			name:     "MultipleUnsortedTags",
			tags:     []int{28, 4, 10},
			expected: "4, 10, 28",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := formatReservedTags(tc.tags)

			assert.Equal(t, tc.expected, actual)
		})
	}
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

func TestComputeProtoMappings(t *testing.T) {
	testCases := []struct {
		param                  Param
		expectedProtoFieldName string
		expectedGoExpression   string
		expectedProtoTag       int
		isSkipped              bool
	}{
		{
			param: Param{
				FlagName:       "app-name",
				ConfigPath:     "app-name",
				Type:           "string",
				ProtoType:      "bool",
				ProtoFieldName: "is_app_name_set",
				ProtoTag:       1,
			},
			expectedProtoFieldName: "IsAppNameSet",
			expectedGoExpression:   `config.AppName != ""`,
			expectedProtoTag:       1,
		},
		{
			param: Param{
				FlagName:       "cache-dir",
				ConfigPath:     "cache-dir",
				Type:           "resolvedPath",
				ProtoType:      "bool",
				ProtoFieldName: "is_cache_dir_set",
				ProtoTag:       2,
			},
			expectedProtoFieldName: "IsCacheDirSet",
			expectedGoExpression:   `string(config.CacheDir) != ""`,
			expectedProtoTag:       2,
		},
		{
			param: Param{
				FlagName:       "file-cache-max-size-mb",
				ConfigPath:     "file-cache.max-size-mb",
				Type:           "int",
				ProtoType:      "sint64",
				ProtoFieldName: "file_cache_max_size_mb",
				ProtoTag:       3,
			},
			expectedProtoFieldName: "FileCacheMaxSizeMb",
			expectedGoExpression:   "config.FileCache.MaxSizeMb",
			expectedProtoTag:       3,
		},
		{
			param: Param{
				FlagName:       "fuse-options",
				ConfigPath:     "file-system.fuse-options",
				Type:           "[]string",
				ProtoType:      "bool",
				ProtoFieldName: "is_file_system_fuse_options_set",
				ProtoTag:       4,
			},
			expectedProtoFieldName: "IsFileSystemFuseOptionsSet",
			expectedGoExpression:   "len(config.FileSystem.FuseOptions) > 0",
			expectedProtoTag:       4,
		},
		{
			param: Param{
				FlagName:       "client-protocol",
				ConfigPath:     "gcs-connection.client-protocol",
				Type:           "protocol",
				ProtoType:      "string",
				ProtoFieldName: "gcs_connection_client_protocol",
				ProtoTag:       5,
			},
			expectedProtoFieldName: "GcsConnectionClientProtocol",
			expectedGoExpression:   "string(config.GcsConnection.ClientProtocol)",
			expectedProtoTag:       5,
		},
		{
			param: Param{
				FlagName:       "machine-type",
				ConfigPath:     "machine-type",
				Type:           "string",
				ProtoType:      "string",
				ProtoFieldName: "machine_type",
				ProtoTag:       6,
			},
			expectedProtoFieldName: "MachineType",
			expectedGoExpression:   "string(config.MachineType)",
			expectedProtoTag:       6,
		},
		{
			param: Param{
				FlagName:       "profile",
				ConfigPath:     "profile",
				Type:           "string",
				ProtoType:      "string",
				ProtoFieldName: "profile",
				ProtoTag:       7,
			},
			expectedProtoFieldName: "Profile",
			expectedGoExpression:   "string(config.Profile)",
			expectedProtoTag:       7,
		},
		{
			param: Param{
				FlagName:       "deprecated-flag",
				ConfigPath:     "",
				Type:           "bool",
				ProtoType:      "",
				ProtoFieldName: "",
				ProtoTag:       0,
			},
			isSkipped: true,
		},
	}
	params := []Param{
		testCases[2].param,
		testCases[0].param,
		testCases[1].param,
		testCases[7].param,
		testCases[3].param,
		testCases[4].param,
		testCases[5].param,
		testCases[6].param,
	}

	mappings, err := computeProtoMappings(params)

	require.NoError(t, err)
	require.Len(t, mappings, 7)
	mappingIdx := 0
	for _, tc := range testCases {
		if tc.isSkipped {
			continue
		}
		assert.Equal(t, tc.expectedProtoFieldName, mappings[mappingIdx].ProtoFieldName)
		assert.Equal(t, tc.expectedGoExpression, mappings[mappingIdx].GoExpression)
		assert.Equal(t, tc.expectedProtoTag, mappings[mappingIdx].ProtoTag)
		mappingIdx++
	}
}
