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

// Provide a helper for directory operations.
package operations

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

const FilePermission_0400 = 0400
const FilePermission_0600 = 0600
const FilePermission_0777 = 0777
const DirPermission_0755 = 0755
const MiB = 1024 * 1024

func executeCommandForOperation(cmd *exec.Cmd) (err error) {
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("command execution %s failed: %v", cmd, cmd.Stderr)
	}
	return
}

func CopyDir(srcDirPath string, destDirPath string) (err error) {
	cmd := exec.Command("cp", "--recursive", srcDirPath, destDirPath)

	err = executeCommandForOperation(cmd)

	return
}

func CopyObject(srcPath string, destPath string) (err error) {
	cmd := exec.Command("cp", srcPath, destPath)

	err = executeCommandForOperation(cmd)

	return
}

func Move(srcPath string, destPath string) (err error) {
	cmd := exec.Command("mv", srcPath, destPath)

	err = executeCommandForOperation(cmd)

	return
}

func RenameDir(dirName string, newDirName string) (err error) {
	if _, err = os.Stat(newDirName); err == nil {
		err = fmt.Errorf("renamed directory %s already present", newDirName)
		return
	}

	if err = os.Rename(dirName, newDirName); err != nil {
		err = fmt.Errorf("rename unsuccessful: %v", err)
		return
	}

	if _, err = os.Stat(dirName); err == nil {
		err = fmt.Errorf("original directory %s still exists", dirName)
		return
	}
	if _, err = os.Stat(newDirName); err != nil {
		err = fmt.Errorf("renamed directory %s not found", newDirName)
		return
	}
	return
}

func CreateDirectoryWithNFiles(numberOfFiles int, dirPath string, prefix string, t *testing.T) {
	// 1. Create the directory.
	err := os.Mkdir(dirPath, FilePermission_0777)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		t.Fatalf("Error in creating directory %q: %v", dirPath, err)
	}

	// Limit the maximum number of I/O goroutines that can run simultaneously.
	const maxConcurrency = 1024
	sem := make(chan struct{}, maxConcurrency)

	// 2. Setup a WaitGroup to manage concurrent Go routines
	var wg sync.WaitGroup

	// 3. Setup a channel to collect and report any errors
	// A buffered channel is used so a Go routine won't block if the main thread
	// has already called t.Fatalf and stopped processing.
	errCh := make(chan error, numberOfFiles)

	// 4. Loop to start concurrent file creation
	for i := 1; i <= numberOfFiles; i++ {
		// ACQUIRE TOKEN: This will block if 1024 goroutines are currently active
		// to prevent thread limit exhaustion.
		sem <- struct{}{}

		wg.Add(1) // Increment the counter for each Go routine started

		// Capture the loop variable locally to avoid race conditions
		// where multiple Go routines might use the final value of i.
		i := i

		go func() {
			defer wg.Done() // Decrement the counter when the Go routine finishes
			// RELEASE TOKEN: Execute this immediately before the goroutine exits
			// to allow the next waiting goroutine to proceed.
			defer func() { <-sem }()

			// Create file with name prefix + i (e.g., temp1, temp2)
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))

			file, err := os.Create(filePath)
			if err != nil {
				// Send the error to the channel instead of calling t.Fatalf directly
				errCh <- err
				return
			}

			// Closing file at the end.
			CloseFileShouldNotThrowError(t, file)
		}()
	}

	// 5. Wait for all Go routines to finish
	wg.Wait()

	// 6. Check for errors
	// We need to check if any errors were sent to the channel.
	select {
	case err := <-errCh:
		// If an error is received, fail the test
		t.Fatalf("Failed to create file during parallel execution: %v", err)
	default:
		// No errors were available, so proceed
	}
}

func RemoveDir(dirPath string) {
	if err := os.RemoveAll(dirPath); err != nil {
		log.Printf("os.RemoveAll(%s): %v", dirPath, err)
	}
}

func ReadDirectory(dirPath string, t *testing.T) (entries []os.DirEntry) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		t.Fatalf("os.ReadDir(%s) err: %v", dirPath, err)
	}
	return
}

func VerifyDirectoryEntry(entry os.DirEntry, dirName string, t *testing.T) {
	if !entry.IsDir() {
		t.Fatalf("Expected: directory entry, Got: file entry.")
	}
	if entry.Name() != dirName {
		t.Fatalf("File name, Expected: %s, Got: %s", dirName, entry.Name())
	}
}

func VerifyCountOfDirectoryEntries(expected, got int, t *testing.T) {
	if expected != got {
		t.Fatalf("directory entry count mismatch, expected: %d, got: %d", expected, got)
	}
}

func CreateDirectory(dirPath string, t testing.TB) {
	err := os.Mkdir(dirPath, DirPermission_0755)

	// Verify MkDir operation succeeds.
	if err != nil {
		t.Fatalf("Error while creating directory, err: %v", err)
	}
}

func DirSizeMiB(dirPath string) (dirSizeMB int64, err error) {
	var totalSize int64
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	dirSizeMB = totalSize / MiB

	return dirSizeMB, err
}

func DeleteManagedFoldersInBucket(managedFolderPath, bucket string) {
	gcloudDeleteManagedFolderCmd := fmt.Sprintf("alpha storage rm -r gs://%s/%s", bucket, managedFolderPath)

	_, err := ExecuteGcloudCommand(gcloudDeleteManagedFolderCmd)
	if err != nil && !strings.Contains(err.Error(), "The following URLs matched no objects or files") {
		log.Fatalf("Error while deleting managed folder: %v", err)
	}
}

func CreateManagedFoldersInBucket(managedFolderPath, bucket string) {
	gcloudCreateManagedFolderCmd := fmt.Sprintf("alpha storage managed-folders create gs://%s/%s", bucket, managedFolderPath)

	_, err := ExecuteGcloudCommand(gcloudCreateManagedFolderCmd)
	if err != nil && !strings.Contains(err.Error(), "The specified managed folder already exists") {
		log.Fatalf("Error while creating managed folder: %v", err)
	}
}

func CopyFileInBucket(srcfilePath, destFilePath, bucket string, t *testing.T) {
	gcloudCopyFileCmd := fmt.Sprintf("alpha storage cp %s gs://%s/%s/", srcfilePath, bucket, destFilePath)

	_, err := ExecuteGcloudCommand(gcloudCopyFileCmd)
	if err != nil {
		t.Fatalf("Error while copying file in bucket: %v", err)
	}
}
