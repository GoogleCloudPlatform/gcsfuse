/*
 * Copyright 2024 Google Inc. All Rights Reserved.
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

package metadata_prefetch

import (
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
)

func configForInfiniteMetadataCache(fileName string) string {
	// Set up config file for file cache.
	mountConfig := config.MountConfig{
		MetadataCacheConfig: config.MetadataCacheConfig{
			TtlInSeconds:       -1,
			TypeCacheMaxSizeMB: -1,
			StatCacheMaxSizeMB: -1,
		},
		LogConfig: config.LogConfig{
			Severity:        config.TRACE,
			Format:          "json",
			FilePath:        setup.LogFile(),
			LogRotateConfig: config.DefaultLogRotateConfig(),
		},
	}
	return setup.YAMLConfigFile(mountConfig, fileName)
}

func TestListWithMetadataPrefetchDisabled(t *testing.T) {
	setup.SetUpTestDir()
	flags := []string{"--only-dir=1000000_objects", "--metadata-prefetch-on-mount=disabled"}
	err := static_mounting.MountGcsfuseForBucket("gargnitin-memory-testing-bucket-20230809", flags)
	if err != nil {
		t.Fatal("Mounting failed")
	}
	defer setup.UnmountGCSFuseAndDeleteLogFile(setup.MntDir())

	objs, err := os.ReadDir(setup.MntDir())

	assert := assert.New(t)
	if assert.Nil(err) {
		assert.Equal(len(objs), 1_000_000)
	}
}

func TestListWithMetadataSynchronousPrefetch(t *testing.T) {
	setup.SetUpTestDir()
	flags := []string{
		"--only-dir=1000000_objects",
		"--metadata-prefetch-on-mount=sync",
		"--config-file=" + configForInfiniteMetadataCache("config.yaml"),
	}
	err := static_mounting.MountGcsfuseForBucket("gargnitin-memory-testing-bucket-20230809", flags)
	if err != nil {
		t.Fatal("Mounting failed")
	}
	defer setup.UnmountGCSFuseAndDeleteLogFile(setup.MntDir())

	objs, err := os.ReadDir(setup.MntDir())

	assert := assert.New(t)
	if assert.Nil(err) {
		assert.Equal(len(objs), 1_000_000)
	}
}

func TestListWithMetadataAsyncPrefetch(t *testing.T) {
	setup.SetUpTestDir()
	flags := []string{
		"--only-dir=1000000_objects",
		"--metadata-prefetch-on-mount=async",
		"--config-file=" + configForInfiniteMetadataCache("config.yaml"),
	}
	err := static_mounting.MountGcsfuseForBucket("gargnitin-memory-testing-bucket-20230809", flags)
	if err != nil {
		t.Fatal("Mounting failed")
	}
	defer setup.UnmountGCSFuseAndDeleteLogFile(setup.MntDir())

	objs, err := os.ReadDir(setup.MntDir())

	assert := assert.New(t)
	if assert.Nil(err) {
		assert.Equal(len(objs), 1_000_000)
	}
}
