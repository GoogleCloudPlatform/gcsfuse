// Copyright 2023 Google LLC
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

// Provide tests for explicit directory only.
package explicit_dir_test

import (
	"context"
	"log"
	"os"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup/implicit_and_explicit_dir_setup"
)

const DirForExplicitDirTests = "dirForExplicitDirTests"

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()
	var storageClient *storage.Client
	// Create storage client before running tests.
	ctx := context.Background()
	storageClient, err := client.CreateStorageClient(ctx)
	if err != nil {
		log.Printf("Error creating storage client: %v\n", err)
		os.Exit(1)
	}

	defer storageClient.Close()

	// These tests will not run on HNS buckets because the "--implicit-dirs=false" flag does not function similarly to how it does on FLAT buckets.
	// Note that HNS buckets do not have the concept of implicit directories.
	if setup.IsHierarchicalBucket(ctx, storageClient) {
		log.Println("These tests will not run on HNS buckets.")
		return
	}

	flags := [][]string{{"--implicit-dirs=false"}}

	if !testing.Short() {
		flags = append(flags, []string{"--client-protocol=grpc", "--implicit-dirs=false"})
	}

	successCode := implicit_and_explicit_dir_setup.RunTestsForImplicitDirAndExplicitDir(flags, m)

	os.Exit(successCode)
}
