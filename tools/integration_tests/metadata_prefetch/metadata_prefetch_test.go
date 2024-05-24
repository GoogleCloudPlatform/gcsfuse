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
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
)

func configForInfiniteMetadataCache(t *testing.T, fileName string) string {
	t.Helper()
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

func list1MFiles(t *testing.T, mode string) {
	err := setup.SetUpTestDir()
	if err != nil {
		t.Fatal("Error occurred while setting up the test dir")
	}
	flags := []string{
		"--only-dir=1000000_objects",
		"--metadata-prefetch-on-mount=" + mode,
		"--config-file=" + configForInfiniteMetadataCache(t, "config.yaml"),
	}
	err = static_mounting.MountGcsfuseForBucket("gcsfuse-metadata-prefetch-test", flags)
	if err != nil {
		t.Fatal("Mounting failed")
	}
	t.Cleanup(func() { setup.UnmountGCSFuseAndDeleteLogFile(setup.MntDir()) })

	objs, err := os.ReadDir(setup.MntDir())

	assert := assert.New(t)
	if assert.Nil(err) {
		assert.Equal(len(objs), 1_000_000)
	}
}

func oneMillionDirOneFileEach(t *testing.T, mode string) {
	err := setup.SetUpTestDir()
	if err != nil {
		t.Fatal("Error occurred while setting up the test dir")
	}
	flags := []string{
		"--only-dir=1M_dir_one_file_each",
		"--metadata-prefetch-on-mount=" + mode,
		"--config-file=" + configForInfiniteMetadataCache(t, "config.yaml"),
	}
	err = static_mounting.MountGcsfuseForBucket("gcsfuse-metadata-prefetch-test-1000dir-1000filesperdir", flags)
	if err != nil {
		t.Fatal("Mounting failed")
	}
	t.Cleanup(func() { setup.UnmountGCSFuseAndDeleteLogFile(setup.MntDir()) })

	baseDir := setup.MntDir()
	objs, err := os.ReadDir(baseDir)

	assert := assert.New(t)
	if assert.Nil(err) {
		assert.Equal(len(objs), 1_000_000)
		for _, d := range objs {
			assert.True(d.IsDir())
			subdir := path.Join(baseDir, d.Name())
			files, err := os.ReadDir(subdir)
			assert.Nil(err)
			assert.Equal(len(files), 1)
			assert.False(files[0].IsDir())
		}
	}
}

func dirs1000With1000FilesEach(t *testing.T, mode string) {
	err := setup.SetUpTestDir()
	if err != nil {
		t.Fatal("Error occurred while setting up the test dir")
	}
	flags := []string{
		"--only-dir=base_dir",
		"--metadata-prefetch-on-mount=" + mode,
		"--implicit-dirs",
		"--config-file=" + configForInfiniteMetadataCache(t, "config.yaml"),
	}
	err = static_mounting.MountGcsfuseForBucket("gcsfuse-metadata-prefetch-test-1000dir-1000filesperdir", flags)
	if err != nil {
		t.Fatal("Mounting failed")
	}
	t.Cleanup(func() { setup.UnmountGCSFuseAndDeleteLogFile(setup.MntDir()) })
	baseDir := setup.MntDir()

	objs, err := os.ReadDir(baseDir)

	assert := assert.New(t)
	if assert.Nil(err) {
		assert.Equal(len(objs), 1_000)
		for _, d := range objs {
			assert.True(d.IsDir())
			subdir := path.Join(baseDir, d.Name())
			files, err := os.ReadDir(subdir)
			assert.Nil(err)
			assert.Equal(len(files), 1_000)
		}
	}
}

func deepNested(t *testing.T, mode string) {
	err := setup.SetUpTestDir()
	if err != nil {
		t.Fatal("Error occurred while setting up the test dir")
	}
	flags := []string{
		"--only-dir=deep_nested_one_file_and_dir_per_level",
		"--metadata-prefetch-on-mount=" + mode,
		"--implicit-dirs",
		"--config-file=" + configForInfiniteMetadataCache(t, "config.yaml"),
	}

	err = static_mounting.MountGcsfuseForBucket("gcsfuse-metadata-prefetch-test-1000dir-1000filesperdir", flags)

	if err != nil {
		t.Fatal("Mounting failed")
	}
	t.Cleanup(func() { setup.UnmountGCSFuseAndDeleteLogFile(setup.MntDir()) })

	baseDir := setup.MntDir()
	assert := assert.New(t)
	idx := 0
	for idx < 11 {
		objs, err := os.ReadDir(baseDir)
		if assert.Nil(err) {
			if idx == 0 {
				assert.Equal(1, len(objs))
				assert.True(objs[0].IsDir())
				idx++
				baseDir = path.Join(baseDir, "a")
				continue
			}
			if idx == 10 {
				assert.Equal(1, len(objs))
				assert.False(objs[0].IsDir())
				break
			}
			assert.Equal(2, len(objs))
			assert.True(objs[0].IsDir() != objs[1].IsDir())
			baseDir = path.Join(baseDir, "a")
			idx++
		}
	}
}

func TestMetadataPrefetch(t *testing.T) {
	setup.IgnoreTestIfIntegrationTestFlagIsNotSet(t)
	if setup.MountedDirectory() != "" {
		t.SkipNow()
	}
	tests := []struct {
		name   string
		testFn func(*testing.T, string)
	}{
		{
			name:   "deep_nested",
			testFn: deepNested,
		},
		{
			name:   "dirs1000With1000FilesEach",
			testFn: dirs1000With1000FilesEach,
		},
		{
			name:   "list1MFiles",
			testFn: list1MFiles,
		},
		{
			name:   "oneMillionDirOneFileEach",
			testFn: oneMillionDirOneFileEach,
		},
	}

	for _, k := range tests {
		for _, m := range []string{"sync", "async", "disabled"} {
			t.Run(k.name+"_"+m, func(t *testing.T) { k.testFn(t, m) })
		}
	}
}
