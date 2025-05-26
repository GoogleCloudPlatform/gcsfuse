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
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/compute/metadata"
	gcpProfiler "google.golang.org/api/cloudprofiler/v2"
	"google.golang.org/api/option"
)

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
// returns true in case of the first match, false if no matching profile found.
// Ref: https://cloud.google.com/profiler/docs/reference/v2/rest
func checkIfProfileExistForServiceAndVersion(
	ctx context.Context,
	t *testing.T, // Pass testing.T for logging within the helper
	profilerAPIClient *gcpProfiler.Service,
	projectID string,
) bool {

	t.Logf("Querying profiles for service [%s] version [%s]", testServiceName, testServiceVersion)

	listCtx, listCancel := context.WithTimeout(ctx, 10*time.Minute)
	defer listCancel()

	pagesFetched := 0
	completedErr := errors.New("completed")
	listCall := profilerAPIClient.Projects.Profiles.List(fmt.Sprintf("projects/%s", projectID))
	err := listCall.Pages(listCtx, func(resp *gcpProfiler.ListProfilesResponse) error {
		pagesFetched++
		t.Logf("Processing page %d of profiles, number of profiles in page: %d", pagesFetched, len(resp.Profiles))
		// Filter by service name and version on the client side.
		for _, p := range resp.Profiles {
			if p.Deployment != nil && p.Deployment.Target == testServiceName {
				profileAPIVersion := ""
				if p.Deployment.Labels != nil {
					profileAPIVersion = p.Deployment.Labels["version"] // "version" is the label key used by the agent
				}
				if profileAPIVersion == testServiceVersion {
					t.Logf("Found matching profile: Type=%s, ID=%s", p.ProfileType, p.Deployment.Labels["version"])
					return completedErr
				}
			}
		}
		return nil // Continue to the next page
	})
	if err != nil && err != completedErr {
		t.Logf("while listing: %v", err)
		return false
	}

	if pagesFetched == 0 {
		t.Logf("No profiles found")
		return false
	}

	return true
}

func TestValidateProfilerWithActualService(t *testing.T) {
	// GCSFuse process will be started as part of mount.
	// Allow some time to export the profile data to GCP profiler service.
	time.Sleep(time.Minute)

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
	if !checkIfProfileExistForServiceAndVersion(apiCtx, t, profilerAPIClient, projectID) {
		t.Errorf("No valid profile found for service [%s] and version [%s]", testServiceName, testServiceVersion)
	}
}
