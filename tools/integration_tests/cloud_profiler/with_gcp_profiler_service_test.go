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

package cloud_profiler_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"cloud.google.com/go/compute/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	gcpProfiler "google.golang.org/api/cloudprofiler/v2"
	"google.golang.org/api/option"
)

type CloudProfilerSuite struct {
	suite.Suite
}

func (s *CloudProfilerSuite) writeSingleRandomFile() error {
	t := s.T()
	data := make([]byte, 100*1024*1024)
	if _, err := rand.Read(data); err != nil {
		return fmt.Errorf("failed to generate random data: %v", err)
	}

	fileName := filepath.Join(setup.MntDir(), fmt.Sprintf("load_file_%d.bin", time.Now().UnixNano()))
	f, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create load file: %v", err)
	}
	defer f.Close()

	if _, err = f.Write(data); err != nil {
		return fmt.Errorf("failed to write to load file %s: %v", fileName, err)
	}
	t.Logf("Successfully wrote 100MB to %s", fileName)
	return nil
}


func getGCPProjectID(t *testing.T) string {
	fetchProjectCtx, fetchProjectCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer fetchProjectCancel()

	projectID, err := metadata.ProjectIDWithContext(fetchProjectCtx) // Reduced timeout, 5s is usually sufficient.
	if err != nil {
		t.Logf("metadata.ProjectIDWithContext failed: %v, try fetching from environment variable.", err)
		projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
		if projectID == "" {
			t.Skip("Not able to fetch project ID from metadata server or GOOGLE_CLOUD_PROJECT environment variable. Skipping integration test.")
		}
	}
	return projectID
}

// checkIfProfileExistForServiceAndVersion queries the Cloud Profiler API for profiles
// returns true just after the first matching profile, false if no matching profile found.
// Ref: https://cloud.google.com/profiler/docs/reference/v2/rest
func checkIfProfileExistForServiceAndVersion(
	ctx context.Context,
	t *testing.T, // Pass testing.T for logging within the helper
	profilerAPIClient *gcpProfiler.Service,
	projectID string,
) (bool, error) {

	t.Logf("Querying profiles for service [%s] with version [%s]", testServiceName, testVersionName)

	listCtx, listCancel := context.WithCancel(ctx)
	defer listCancel()

	pagesFetched := 0
	profileFound := false
	listCall := profilerAPIClient.Projects.Profiles.List(fmt.Sprintf("projects/%s", projectID))
	err := listCall.Pages(listCtx, func(resp *gcpProfiler.ListProfilesResponse) error {
		pagesFetched++
		t.Logf("Processing page %d of profiles, number of profiles in page: %d", pagesFetched, len(resp.Profiles))
		for _, p := range resp.Profiles {
			if p.Deployment == nil || p.Deployment.Labels == nil {
				continue
			}
			// Break early as test service name is lexicographically smaller than any other service name.
			if p.Deployment.Target > testServiceName {
				return fmt.Errorf("Didn't find matching profile after fetching %d pages, profile not yet available.", pagesFetched)
			}
			// Return if matching profile found.
			if p.Deployment.Target == testServiceName && p.Deployment.Labels["version"] == testVersionName {
				t.Logf("Found matching profile: Type=%s, ServiceName=%s, Version=%s", p.ProfileType, p.Deployment.Target, p.Deployment.Labels["version"])
				profileFound = true
				return fmt.Errorf("Returning error on success to break pagination early")
			}
		}
		return nil
	})
	if profileFound {
		return true, nil
	}
	return false, err
}

func (s *CloudProfilerSuite) TestValidateProfilerWithActualService() {
	t := s.T()
	// 1. Fetch GCP projectID.
	// 2. Create a profiler service api client.
	// 3. Make list call to the profiler service api client and fetch the profiles.
	// 4. Filter and match if the right profile exists.
	projectID := getGCPProjectID(t)
	apiCtx := context.Background()
	profilerAPIClient, err := gcpProfiler.NewService(apiCtx, option.WithScopes(gcpProfiler.CloudPlatformScope))
	if err != nil {
		t.Fatalf("Failed to create Cloud Profiler API client: %v", err)
	}
	t.Logf("Waiting for cloud profile to eventually appear for service [%s] and version [%s]", testServiceName, testVersionName)
	operations.RetryUntil(apiCtx, t, retryFrequency, retryDuration, func() (bool, error) {
		if err := s.writeSingleRandomFile(); err != nil {
			t.Logf("Failed to write load file: %v. So profile generation may be affected...", err)
		}
		return checkIfProfileExistForServiceAndVersion(apiCtx, t, profilerAPIClient, projectID)
	})
}

func TestCloudProfilerSuite(t *testing.T) {
	suite.Run(t, new(CloudProfilerSuite))
}
