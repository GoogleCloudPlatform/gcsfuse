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

package benchmark

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

// RunPreflight performs connectivity and permission checks before the main
// benchmark or prepare run executes. It always prints to stdout regardless of
// the -v verbosity level, since "will this work?" is always useful feedback.
//
// All modes: LIST the object prefix to verify connection and credentials.
//
//	If the prefix is empty and mode is "benchmark", a warning is printed
//	(reads will fail). Execution continues — the caller decides whether to abort.
//
// Prepare mode additionally: PUT a small sentinel object, LIST to confirm it
// is visible, GET to verify content round-trips correctly, then DELETE it.
// DELETE failures are reported as warnings, not errors.
//
// Returns a non-nil error only when a condition is detected that will
// certainly cause the test to fail (cannot list, cannot write in prepare mode).
func RunPreflight(ctx context.Context, bucket gcs.Bucket, bucketName, prefix, mode string, out io.Writer) error {
	isPrepare := strings.ToLower(mode) == "prepare"
	totalChecks := 1
	if isPrepare {
		totalChecks = 5
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "=== Pre-flight check ===")
	fmt.Fprintln(out)

	// ------------------------------------------------------------------
	// Check 1: LIST the prefix — verifies connectivity and credentials.
	// ------------------------------------------------------------------
	displayPath := fmt.Sprintf("gs://%s/%s", bucketName, prefix)
	fmt.Fprintf(out, "  [%d/%d] LIST %s ... ", 1, totalChecks, displayPath)

	listStart := time.Now()
	listing, err := bucket.ListObjects(ctx, &gcs.ListObjectsRequest{
		Prefix:     prefix,
		MaxResults: listMaxResults,
	})
	listElapsed := time.Since(listStart)

	if err != nil {
		fmt.Fprintf(out, "FAIL [%s]\n", listElapsed.Round(time.Millisecond))
		fmt.Fprintf(out, "         Error: %v\n", err)
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  Pre-flight: FAILED — cannot list bucket prefix. Check credentials and bucket name.")
		fmt.Fprintln(out)
		return fmt.Errorf("pre-flight LIST failed: %w", err)
	}

	count := 0
	hasMore := false
	if listing != nil {
		count = len(listing.MinObjects)
		hasMore = listing.ContinuationToken != ""
	}
	countStr := fmt.Sprintf("%d", count)
	if hasMore {
		countStr = fmt.Sprintf("%d+", count)
	}

	switch {
	case count == 0 && !isPrepare:
		fmt.Fprintf(out, "OK [%s] — WARNING: prefix is EMPTY\n", listElapsed.Round(time.Millisecond))
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  Pre-flight: WARNING — no objects found at prefix; bench reads will likely fail.")
		fmt.Fprintln(out, "  Proceeding anyway (prefix may be wrong, or objects live under a sub-prefix).")
		fmt.Fprintln(out)
		return nil
	case count == 0 && isPrepare:
		fmt.Fprintf(out, "OK [%s] — prefix is EMPTY (ready for prepare)\n", listElapsed.Round(time.Millisecond))
	default:
		fmt.Fprintf(out, "OK [%s] — %s object(s) found\n", listElapsed.Round(time.Millisecond), countStr)
		if isPrepare && count > 0 {
			fmt.Fprintf(out, "         Note: prepare will OVERWRITE the %s existing object(s) at this prefix.\n", countStr)
		}
	}

	if !isPrepare {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  Pre-flight: PASSED — benchmark should work.")
		fmt.Fprintln(out)
		return nil
	}

	// ------------------------------------------------------------------
	// Prepare-mode checks: PUT / LIST / GET / DELETE a sentinel object.
	// This verifies full read/write/delete permissions before we start
	// writing potentially thousands of objects.
	// ------------------------------------------------------------------
	sentinel := prefix + "_gcsbench_preflight_"
	payload := []byte("gcsfuse-bench pre-flight sentinel ok")

	// Safety net: clean up the sentinel on return even if an early step fails.
	defer func() {
		_ = bucket.DeleteObject(ctx, &gcs.DeleteObjectRequest{Name: sentinel})
	}()

	// --- Check 2: PUT ---
	fmt.Fprintf(out, "  [%d/%d] PUT  %s_gcsbench_preflight_ ... ", 2, totalChecks, prefix)
	writeStart := time.Now()
	_, err = bucket.CreateObject(ctx, &gcs.CreateObjectRequest{
		Name:     sentinel,
		Contents: io.NopCloser(bytes.NewReader(payload)),
	})
	writeElapsed := time.Since(writeStart)

	if err != nil {
		fmt.Fprintf(out, "FAIL [%s]\n", writeElapsed.Round(time.Millisecond))
		fmt.Fprintf(out, "         Error: %v\n", err)
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  Pre-flight: FAILED — cannot write objects. Prepare will fail.")
		fmt.Fprintln(out)
		return fmt.Errorf("pre-flight PUT failed: %w", err)
	}
	fmt.Fprintf(out, "OK [%s]\n", writeElapsed.Round(time.Millisecond))

	// --- Check 3: LIST sentinel to confirm visibility ---
	fmt.Fprintf(out, "  [%d/%d] LIST %s_gcsbench_preflight_ ... ", 3, totalChecks, prefix)
	verifyStart := time.Now()
	verifyListing, verifyErr := bucket.ListObjects(ctx, &gcs.ListObjectsRequest{
		Prefix:     sentinel,
		MaxResults: 10,
	})
	verifyElapsed := time.Since(verifyStart)

	switch {
	case verifyErr != nil:
		fmt.Fprintf(out, "FAIL [%s]: %v\n", verifyElapsed.Round(time.Millisecond), verifyErr)
	case verifyListing == nil || len(verifyListing.MinObjects) == 0:
		fmt.Fprintf(out, "WARNING [%s] — object not yet visible (eventual consistency?)\n",
			verifyElapsed.Round(time.Millisecond))
	default:
		fmt.Fprintf(out, "OK [%s] — object visible\n", verifyElapsed.Round(time.Millisecond))
	}

	// --- Check 4: GET and verify content ---
	fmt.Fprintf(out, "  [%d/%d] GET  %s_gcsbench_preflight_ ... ", 4, totalChecks, prefix)
	getStart := time.Now()
	reader, getErr := bucket.NewReaderWithReadHandle(ctx, &gcs.ReadObjectRequest{
		Name: sentinel,
		Range: &gcs.ByteRange{
			Start: 0,
			Limit: uint64(len(payload)) + 16, // a little beyond payload length
		},
	})

	if getErr != nil {
		getElapsed := time.Since(getStart)
		fmt.Fprintf(out, "FAIL [%s]: %v\n", getElapsed.Round(time.Millisecond), getErr)
		fmt.Fprintf(out, "         WARNING: wrote object but cannot read it back.\n")
	} else {
		buf, readErr := io.ReadAll(reader)
		reader.Close()
		getElapsed := time.Since(getStart)

		switch {
		case readErr != nil:
			fmt.Fprintf(out, "FAIL [%s] (read error): %v\n", getElapsed.Round(time.Millisecond), readErr)
		case !bytes.Equal(buf, payload):
			fmt.Fprintf(out, "FAIL [%s] — content mismatch (got %d bytes, expected %d)\n",
				getElapsed.Round(time.Millisecond), len(buf), len(payload))
		default:
			fmt.Fprintf(out, "OK [%s] — %d bytes, content verified\n",
				getElapsed.Round(time.Millisecond), len(buf))
		}
	}

	// --- Check 5: DELETE ---
	fmt.Fprintf(out, "  [%d/%d] DELETE %s_gcsbench_preflight_ ... ", 5, totalChecks, prefix)
	delStart := time.Now()
	delErr := bucket.DeleteObject(ctx, &gcs.DeleteObjectRequest{Name: sentinel})
	delElapsed := time.Since(delStart)

	if delErr != nil {
		fmt.Fprintf(out, "WARNING [%s]: %v\n", delElapsed.Round(time.Millisecond), delErr)
		fmt.Fprintf(out, "         (delete permission not granted — clean up %q manually)\n", sentinel)
	} else {
		fmt.Fprintf(out, "OK [%s]\n", delElapsed.Round(time.Millisecond))
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "  Pre-flight: PASSED — prepare should work.")
	fmt.Fprintln(out)
	return nil
}
