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
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/compute/metadata"
	gcpProfiler "google.golang.org/api/cloudprofiler/v2"
	"google.golang.org/api/option"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

func performRepeatedRead(ctx context.Context, t *testing.T, filePath string, duration time.Duration) {
	timeout := time.After(duration)
	readFileCnt := 0
	for {
		select {
		case <-timeout:
			t.Logf("performRepeatedRead: Duration %v reached.", duration)
			return
		case <-ctx.Done():
			t.Log("performRepeatedRead: Context cancelled.")
			return
		default:
			_, err := operations.ReadFile(filePath)
			if err != nil { // Log transient errors but continue reading
				t.Logf("ReadFile failed during performRepeatedRead (filePath: %s): %v", filePath, err)
			} else {
				readFileCnt++
				// Just to show some progress.
				if readFileCnt%2500 == 0 {
					t.Logf("Read file operation completed %d times.", readFileCnt)
				}
			}
		}
	}
}

// listProfilesForServiceAndVersion queries the Cloud Profiler API for profiles
// matching the given service name and version within the specified time window.
// It handles pagination and basic retries for transient errors.
// Ref: https://cloud.google.com/profiler/docs/reference/v2/rest
func listProfilesForServiceAndVersion(
	ctx context.Context,
	t *testing.T, // Pass testing.T for logging within the helper
	profilerAPIClient *gcpProfiler.Service,
	projectID string,
) ([]*gcpProfiler.Profile, error) {

	t.Logf("Querying profiles for service [%s] version [%s]", testServiceName, testServiceVersion)
	var profiles []*gcpProfiler.Profile
	maxPagesToFetch := 5 // Limiting total list calls, 1000 per calls.
	pagesFetched := 0
	var pageToken string
	for {
		pagesFetched++
		if pagesFetched > maxPagesToFetch {
			t.Logf("Reached max pages (%d) to fetch. Stopping profile query.", maxPagesToFetch)
			break
		}

		listCall := profilerAPIClient.Projects.Profiles.List(fmt.Sprintf("projects/%s", projectID))
		if pageToken != "" {
			listCall.PageToken(pageToken)
		}
		resp, errCall := listCall.Do()
		if errCall != nil {
			// Basic retry for transient errors
			if strings.Contains(errCall.Error(), "try again") || strings.Contains(errCall.Error(), "unavailable") {
				t.Logf("List profiles call failed, retrying once: %v", errCall)
				time.Sleep(10 * time.Second)
				resp, errCall = listCall.Do() // Retry the call
			}
			if errCall != nil {
				return nil, fmt.Errorf("failed to list profiles from API: %w", errCall)
			}
		}

		t.Logf("Size of the response: %d", len(resp.Profiles))
		// Filter by service name and version on the client side.
		for _, p := range resp.Profiles {
			if p.Deployment != nil && p.Deployment.Target == testServiceName {
				profileAPIVersion := ""
				if p.Deployment.Labels != nil {
					profileAPIVersion = p.Deployment.Labels["version"] // "version" is the label key used by the agent
				}
				if profileAPIVersion == testServiceVersion {
					profiles = append(profiles, p)
				}
			}
		}

		if resp.NextPageToken == "" {
			break // No more pages
		}
		pageToken = resp.NextPageToken
		t.Logf("Fetching next page of profiles (token: %s)...", pageToken)
	}

	return profiles, nil
}

func TestValidateProfilerWithActualService(t *testing.T) {
	// Setup directory and file to perform read workload.
	randomData, err := operations.GenerateRandomData(5 * 1024 * 1024)
	if err != nil {
		t.Fatalf("operations.GenerateRandomData: %v", err)
	}
	testDirPath := path.Join(setup.MntDir(), testDirName)
	filePath := path.Join(testDirPath, "a.txt")
	setup.SetupTestDirectory(testDirName)
	operations.CreateFileWithContent(filePath, 0644, string(randomData), t)
	// Start Workload and Allow Time for Profiling
	performRepeatedRead(t.Context(), t, filePath, 3*time.Minute)
	time.Sleep(time.Minute)

	// 1. Fetch GCP projectID.
	// 2. Create a profiler service api client.
	// 3. Make list call to the profiler service api client and fetch the profiles.
	// 4. Filter and match if the right profile data exists.
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
	apiCtx := context.Background()
	profilerAPIClient, err := gcpProfiler.NewService(apiCtx, option.WithScopes(gcpProfiler.CloudPlatformScope))
	if err != nil {
		t.Fatalf("Failed to create Cloud Profiler API client: %v", err)
	}
	profilesFoundForTestServiceAndVersion, err := listProfilesForServiceAndVersion(apiCtx, t, profilerAPIClient, projectID)
	if err != nil {
		t.Fatalf("listProfilesForServiceAndVersion: %v", err)
	}

	// Validate the result.
	expectedProfileTypesInAPI := map[string]bool{
		"CPU":        false,
		"HEAP":       false,
		"THREADS":    false,
		"CONTENTION": false,
		"HEAP_ALLOC": false,
	}
	for _, p := range profilesFoundForTestServiceAndVersion {
		if p.ProfileType != "" {
			if _, ok := expectedProfileTypesInAPI[p.ProfileType]; ok {
				expectedProfileTypesInAPI[p.ProfileType] = true
				t.Logf("Marked API profile type '%s' as found.", p.ProfileType)
			}
		}
	}
	atleastOneFound := false
	allExpectedFound := true
	for typeStr, found := range expectedProfileTypesInAPI {
		if !found {
			allExpectedFound = false
			t.Logf("Expected profile type '%s' was NOT found.", typeStr)
		} else {
			atleastOneFound = true
			t.Logf("Found profile type '%s'.", typeStr)
		}
	}
	if !allExpectedFound {
		t.Logf("Total profiles found for this service/version during test: %d", len(profilesFoundForTestServiceAndVersion))
		if len(profilesFoundForTestServiceAndVersion) > 0 {
			t.Log("Details of profiles found:")
			for _, p := range profilesFoundForTestServiceAndVersion {
				t.Logf("  - Name: %s, Type: %s, Labels: %v", p.Name, p.ProfileType, p.Deployment.Labels)
			}
		}
	} else {
		t.Log("All expected profile types (CPU, HEAP) were successfully found for the test service and version.")
	}
	if !atleastOneFound {
		t.Errorf("Failed: none of the profile found.")
	}
}
