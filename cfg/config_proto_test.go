// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cfg

import (
	"encoding/base64"
	"reflect"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestToProto_Nil(t *testing.T) {
	assert.Nil(t, ToProto(nil))
}

func TestSerializeConfigToProtoBase64_Nil(t *testing.T) {
	str, err := SerializeConfigToProtoBase64(nil)
	assert.NoError(t, err)
	assert.Empty(t, str)
}

func TestSerializeConfigToProtoBase64_DefaultConfig(t *testing.T) {
	config := &Config{}
	str, err := SerializeConfigToProtoBase64(config)
	assert.NoError(t, err)
	assert.Empty(t, str)
}

func TestToProto_DefaultConfig(t *testing.T) {
	config := &Config{}
	protoConfig := ToProto(config)
	require.NotNil(t, protoConfig)

	assert.False(t, protoConfig.IsAppNameSet)
	assert.False(t, protoConfig.IsCacheDirSet)
	assert.False(t, protoConfig.CloudProfilerAllocatedHeap)
	assert.False(t, protoConfig.CloudProfilerEnabled)
	assert.Equal(t, int64(0), protoConfig.FileCacheDownloadChunkSizeMb)
	assert.Equal(t, "", protoConfig.GcsConnectionClientProtocol)
}

func TestToProto_PopulatedFields(t *testing.T) {
	config := &Config{
		AppName:     "custom-app",
		CacheDir:    "/tmp/custom_cache",
		MachineType: "a3-highgpu-8g",
		Profile:     "aiml-training",
		CloudProfiler: CloudProfilerConfig{
			AllocatedHeap: true,
			Cpu:           true,
			Enabled:       true,
			Label:         "test-label",
		},
		FileCache: FileCacheConfig{
			DownloadChunkSizeMb: 64,
			EnableCrc:           true,
			MaxSizeMb:           1024,
		},
		FileSystem: FileSystemConfig{
			DirMode:     0755,
			FuseOptions: []string{"ro", "allow_other"},
		},
		GcsConnection: GcsConnectionConfig{
			ClientProtocol: "grpc",
			BillingProject: "my-gcp-project",
		},
		Write: WriteConfig{
			BlockSizeMb:        32.0,
			EnableRapidAppends: true,
			GlobalMaxBlocks:    100,
		},
	}

	protoConfig := ToProto(config)
	require.NotNil(t, protoConfig)

	// Boolean presence flags for high-risk text/paths
	assert.True(t, protoConfig.IsAppNameSet)
	assert.True(t, protoConfig.IsCacheDirSet)
	assert.True(t, protoConfig.IsCloudProfilerLabelSet)
	assert.True(t, protoConfig.IsFileSystemFuseOptionsSet)
	assert.True(t, protoConfig.IsGcsConnectionBillingProjectSet)

	// Whitelisted string fields
	assert.Equal(t, "a3-highgpu-8g", protoConfig.MachineType)
	assert.Equal(t, "aiml-training", protoConfig.Profile)

	// Explicit values
	assert.True(t, protoConfig.CloudProfilerAllocatedHeap)
	assert.True(t, protoConfig.CloudProfilerCpu)
	assert.True(t, protoConfig.CloudProfilerEnabled)
	assert.Equal(t, int64(64), protoConfig.FileCacheDownloadChunkSizeMb)
	assert.True(t, protoConfig.FileCacheEnableCrc)
	assert.Equal(t, int64(1024), protoConfig.FileCacheMaxSizeMb)
	assert.Equal(t, int32(0755), protoConfig.FileSystemDirMode)
	assert.Equal(t, "grpc", protoConfig.GcsConnectionClientProtocol)
	assert.Equal(t, 32.0, protoConfig.WriteBlockSizeMb)
	assert.True(t, protoConfig.WriteEnableRapidAppends)
	assert.Equal(t, int64(100), protoConfig.WriteGlobalMaxBlocks)
}

func TestSerializeConfigToProtoBase64_RoundTrip(t *testing.T) {
	config := &Config{
		AppName:  "gcsfuse-telemetry-test",
		CacheDir: "/var/cache/gcsfuse",
		CloudProfiler: CloudProfilerConfig{
			Enabled: true,
		},
		FileCache: FileCacheConfig{
			MaxSizeMb: 512,
		},
		GcsConnection: GcsConnectionConfig{
			ClientProtocol:    "http1",
			HttpClientTimeout: 30 * time.Second,
		},
	}

	encoded, err := SerializeConfigToProtoBase64(config)
	require.NoError(t, err)
	assert.NotEmpty(t, encoded)

	decodedBytes, err := base64.RawURLEncoding.DecodeString(encoded)
	require.NoError(t, err)

	var decodedProto pb.Config
	err = proto.Unmarshal(decodedBytes, &decodedProto)
	require.NoError(t, err)

	expectedProto := ToProto(config)
	assert.True(t, proto.Equal(expectedProto, &decodedProto))
}

// populateNonZero recursively fills every field of a struct with non-zero dummy values.
func populateNonZero(v reflect.Value) {
	if !v.CanSet() {
		return
	}
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(42)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(42)
	case reflect.Float32, reflect.Float64:
		v.SetFloat(42.5)
	case reflect.String:
		v.SetString("dummy-test-string")
	case reflect.Slice:
		slice := reflect.MakeSlice(v.Type(), 1, 1)
		populateNonZero(slice.Index(0))
		v.Set(slice)
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			populateNonZero(v.Index(i))
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			populateNonZero(v.Field(i))
		}
	case reflect.Pointer:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		populateNonZero(v.Elem())
	}
}

// TestToProto_AutomatedReflection recursively populates all fields of Config with non-zero values,
// converts it to pb.Config, and uses Protobuf reflection to ensure every generated field in pb.Config
// has received a non-zero value.
func TestToProto_AutomatedReflection(t *testing.T) {
	var config Config
	populateNonZero(reflect.ValueOf(&config).Elem())

	protoConfig := ToProto(&config)
	require.NotNil(t, protoConfig)

	m := protoConfig.ProtoReflect()
	fields := m.Descriptor().Fields()
	require.Greater(t, fields.Len(), 0, "expected proto fields in descriptor")

	var unsetFields []string
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if !m.Has(fd) {
			unsetFields = append(unsetFields, string(fd.Name()))
		}
	}

	assert.Empty(t, unsetFields, "The following proto fields were not populated from Config: %v", unsetFields)
}
