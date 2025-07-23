// Copyright 2025 Google LLC
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

package rapid_appends

import (
	"fmt"
	"log"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *CommonAppendsSuite) TestAppendsAndReads() {
	const metadataCacheTTLSecs = 10
	metadataCacheEnableFlag := fmt.Sprintf("%s%v", "--metadata-cache-ttl-secs=", metadataCacheTTLSecs)
	fileCacheDirFlag := func() string {
		return "--cache-dir=" + getNewEmptyCacheDir(t.primaryMount.rootDir)
	}

	testCases := []struct {
		name string
	}{
		{
			name: "SequentialRead",
		},
	}

	for _, scenario := range []struct {
		enableMetadataCache bool
		enableFileCache     bool
		flags               []string
	}{{
		// all cache disabled
		enableMetadataCache: false,
		enableFileCache:     false,
		flags:               []string{"--write-experimental-enable-rapid-appends=true", "--write-global-max-blocks=-1", "--metadata-cache-ttl-secs=0"},
	}, {
		enableMetadataCache: true,
		enableFileCache:     false,
		flags:               []string{"--write-experimental-enable-rapid-appends=true", "--write-global-max-blocks=-1", metadataCacheEnableFlag},
	}, {
		enableMetadataCache: true,
		enableFileCache:     true,
		flags:               []string{"--write-experimental-enable-rapid-appends=true", "--write-global-max-blocks=-1", metadataCacheEnableFlag, "--file-cache-max-size-mb=-1", fileCacheDirFlag()},
	}, {
		enableMetadataCache: false,
		enableFileCache:     true,
		flags:               []string{"--write-experimental-enable-rapid-appends=true", "--write-global-max-blocks=-1", "--metadata-cache-ttl-secs=0", "--file-cache-max-size-mb=-1", fileCacheDirFlag()},
	}} {
		func() {
			t.mountPrimaryMount(scenario.flags)
			defer t.unmountPrimaryMount()

			log.Printf("Running tests with flags: %v", scenario.flags)

			for _, tc := range testCases {
				t.Run(tc.name, func() {
					// Initially create an unfinalized object.
					t.createUnfinalizedObject()
					defer t.deleteUnfinalizedObject()

					// Open this object as a file for appending on the appropriate mount.
					appendFileHandle := operations.OpenFileInMode(t.T(), path.Join(t.appendMountPath, t.fileName), os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT)
					defer operations.CloseFileShouldNotThrowError(t.T(), appendFileHandle)

					readPath := path.Join(t.primaryMount.testDirPath, t.fileName)
					for i := range numAppends {
						sizeBeforeAppend := len(t.fileContent)
						t.appendToFile(appendFileHandle, setup.GenerateRandomString(appendSize))
						sizeAfterAppend := len(t.fileContent)

						log.Printf("Reading %q right after append#%v ...\n", readPath, i)
						gotContent, err := operations.ReadFile(readPath)
						log.Printf("... finished reading %q right after append#%v\n", readPath, i)

						require.NoError(t.T(), err)
						readContent := string(gotContent)
						// If metadata cache is enabled, gcsfuse reads up to the cached file size.
						// For same-mount appends/reads, file size is always current.
						// The initial read (i=0) bypasses cache, seeing the latest file size.
						if !scenario.enableMetadataCache || !t.isSyncNeededAfterAppend || (i == 0) {
							assert.Equalf(t.T(), t.fileContent, readContent, "failed to match full content in non-metadata-cache/single-mount after %v appends", i+1)
						} else {
							// Read only up to the cached file size (before append).
							assert.Equalf(t.T(), t.fileContent[:sizeBeforeAppend], readContent, "failed to match partial content in metadata-cache dual-mount after %v appends", i+1)

							log.Printf("Sleeping for %v seconds to let metadata cache to expire ...", metadataCacheTTLSecs)
							// Wait for metadata cache to expire to fetch the latest size for the next read.
							time.Sleep(time.Duration(metadataCacheTTLSecs) * time.Second) // Wait for metadata cache to get expired.
							log.Printf("Reading %q right after sleep ...\n", readPath)
							gotContent, err = operations.ReadFile(readPath)
							log.Printf("... finished reading %q right after sleep\n", readPath)

							// Expect read up to the latest file size which is the size after the append.
							require.NoError(t.T(), err)
							readContent = string(gotContent)
							assert.Equalf(t.T(), t.fileContent[:sizeAfterAppend], readContent, "failed to match full content in metadata-cache dual-mount after %v appends", i+1)
						}
					}
				})
			}
		}()
	}
}
