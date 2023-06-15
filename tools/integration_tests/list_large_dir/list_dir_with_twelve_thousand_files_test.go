// Copyright 2023 Google Inc. All Rights Reserved.
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

package list_large_dir_test

import (
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func createTwelveThousandFiles(numberOfFiles int, dirPath string, prefix string, t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for i := 1; i <= 1000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			log.Printf("In 1 thread: %v", filePath)
			_, err := os.Create(filePath)
			if err != nil {
				t.Errorf("Create file at %q: %v", dirPath, err)
			}
		}
	}()

	var wg2 sync.WaitGroup
	wg2.Add(1)

	go func() {
		defer wg2.Done()
		for i := 1001; i <= 2000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			log.Printf("In 2 thread: %v", filePath)
			_, err := os.Create(filePath)
			if err != nil {
				t.Errorf("Create file at %q: %v", dirPath, err)
			}
		}
	}()

	var wg3 sync.WaitGroup
	wg3.Add(1)

	go func() {
		defer wg3.Done()
		for i := 2001; i <= 3000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			log.Printf("In 3 thread: %v", filePath)
			_, err := os.Create(filePath)
			if err != nil {
				t.Errorf("Create file at %q: %v", dirPath, err)
			}
		}
	}()

	var wg4 sync.WaitGroup
	wg4.Add(1)

	go func() {
		defer wg4.Done()
		for i := 3001; i <= 4000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			log.Printf("In 4 thread: %v", filePath)
			_, err := os.Create(filePath)
			if err != nil {
				t.Errorf("Create file at %q: %v", dirPath, err)
			}
		}
	}()

	var wg5 sync.WaitGroup
	wg5.Add(1)

	go func() {
		defer wg5.Done()
		for i := 4001; i <= 5000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			log.Printf("In 5 thread: %v", filePath)
			_, err := os.Create(filePath)
			if err != nil {
				t.Errorf("Create file at %q: %v", dirPath, err)
			}
		}
	}()

	var wg6 sync.WaitGroup
	wg6.Add(1)

	go func() {
		defer wg6.Done()
		for i := 5001; i <= 6000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			log.Printf("In 6 thread: %v", filePath)
			_, err := os.Create(filePath)
			if err != nil {
				t.Errorf("Create file at %q: %v", dirPath, err)
			}
		}
	}()

	var wg7 sync.WaitGroup
	wg7.Add(1)

	go func() {
		defer wg7.Done()
		for i := 6001; i <= 7000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			log.Printf("In 7 thread: %v", filePath)
			_, err := os.Create(filePath)
			if err != nil {
				t.Errorf("Create file at %q: %v", dirPath, err)
			}
		}
	}()

	var wg8 sync.WaitGroup
	wg8.Add(1)

	go func() {
		defer wg8.Done()
		for i := 7001; i <= 8000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			log.Printf("In 8 thread: %v", filePath)
			_, err := os.Create(filePath)
			if err != nil {
				t.Errorf("Create file at %q: %v", dirPath, err)
			}
		}
	}()

	var wg9 sync.WaitGroup
	wg9.Add(1)

	go func() {
		defer wg9.Done()
		for i := 8001; i <= 9000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			log.Printf("In 9 thread: %v", filePath)
			_, err := os.Create(filePath)
			if err != nil {
				t.Errorf("Create file at %q: %v", dirPath, err)
			}
		}
	}()

	var wg10 sync.WaitGroup
	wg10.Add(1)

	go func() {
		defer wg10.Done()
		for i := 9001; i <= 10000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			log.Printf("In 10 thread: %v", filePath)
			_, err := os.Create(filePath)
			if err != nil {
				t.Errorf("Create file at %q: %v", dirPath, err)
			}
		}
	}()

	var wg11 sync.WaitGroup
	wg11.Add(1)

	go func() {
		defer wg11.Done()
		for i := 10001; i <= 11000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			log.Printf("In 11 thread: %v", filePath)
			_, err := os.Create(filePath)
			if err != nil {
				t.Errorf("Create file at %q: %v", dirPath, err)
			}
		}
	}()

	var wg12 sync.WaitGroup
	wg12.Add(1)

	go func() {
		defer wg12.Done()
		for i := 11001; i <= 12000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			log.Printf("In 12 thread: %v", filePath)
			_, err := os.Create(filePath)
			if err != nil {
				t.Errorf("Create file at %q: %v", dirPath, err)
			}
		}
	}()

	wg.Wait()
	wg2.Wait()
	wg3.Wait()
	wg4.Wait()
	wg5.Wait()
	wg6.Wait()
	wg7.Wait()
	wg8.Wait()
	wg9.Wait()
	wg10.Wait()
	wg11.Wait()
	wg12.Wait()
}

func deleteTwelveThousandFiles(numberOfFiles int, dirPath string, prefix string, t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for i := 1; i <= 1000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			os.RemoveAll(filePath)
		}
	}()

	var wg2 sync.WaitGroup
	wg2.Add(1)

	go func() {
		defer wg2.Done()
		for i := 1001; i <= 2000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			os.RemoveAll(filePath)
		}
	}()

	var wg3 sync.WaitGroup
	wg3.Add(1)

	go func() {
		defer wg3.Done()
		for i := 2001; i <= 3000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			os.RemoveAll(filePath)
		}
	}()

	var wg4 sync.WaitGroup
	wg4.Add(1)

	go func() {
		defer wg4.Done()
		for i := 3001; i <= 4000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			os.RemoveAll(filePath)
		}
	}()

	var wg5 sync.WaitGroup
	wg5.Add(1)

	go func() {
		defer wg5.Done()
		for i := 4001; i <= 5000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			os.RemoveAll(filePath)
		}
	}()

	var wg6 sync.WaitGroup
	wg6.Add(1)

	go func() {
		defer wg6.Done()
		for i := 5001; i <= 6000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			os.RemoveAll(filePath)
		}
	}()

	var wg7 sync.WaitGroup
	wg7.Add(1)

	go func() {
		defer wg7.Done()
		for i := 6001; i <= 7000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			os.RemoveAll(filePath)
		}
	}()

	var wg8 sync.WaitGroup
	wg8.Add(1)

	go func() {
		defer wg8.Done()
		for i := 7001; i <= 8000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			os.RemoveAll(filePath)
		}
	}()

	var wg9 sync.WaitGroup
	wg9.Add(1)

	go func() {
		defer wg9.Done()
		for i := 8001; i <= 9000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			os.RemoveAll(filePath)
		}
	}()

	var wg1 sync.WaitGroup
	wg1.Add(1)

	go func() {
		defer wg1.Done()
		for i := 8001; i <= 9000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			os.RemoveAll(filePath)
		}
	}()

	var wg10 sync.WaitGroup
	wg10.Add(1)

	go func() {
		defer wg10.Done()
		for i := 9001; i <= 10000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			os.RemoveAll(filePath)
		}
	}()

	var wg11 sync.WaitGroup
	wg11.Add(1)

	go func() {
		defer wg11.Done()
		for i := 10001; i <= 11000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			os.RemoveAll(filePath)
		}
	}()

	var wg12 sync.WaitGroup
	wg12.Add(1)

	go func() {
		defer wg12.Done()
		for i := 11001; i <= 12000; i++ {
			filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
			os.RemoveAll(filePath)
		}
	}()

	wg.Wait()
	wg1.Wait()
	wg2.Wait()
	wg3.Wait()
	wg4.Wait()
	wg5.Wait()
	wg6.Wait()
	wg7.Wait()
	wg8.Wait()
	wg9.Wait()
	wg10.Wait()
	wg11.Wait()
	wg12.Wait()
}
func checkIfObjNameIsCorrect(objName string, prefix string, maxNumber int, t *testing.T) {
	// Extracting object number.
	objNumberStr := strings.ReplaceAll(objName, prefix, "")
	objNumber, err := strconv.Atoi(objNumberStr)
	if err != nil {
		t.Errorf("Error in extracting file number.")
	}
	if objNumber < 1 || objNumber > maxNumber {
		t.Errorf("Correct object does not exist.")
	}
}

// Test with a bucket with twelve thousand files.
func TestDirectoryWithTwelveThousandFiles(t *testing.T) {
	// Create twelve thousand files in the directoryWithTwelveThousandFiles directory.
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFiles)
	err := os.Mkdir(dirPath, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Error in creating directory: %v", err)
	}
	createTwelveThousandFiles(NumberOfFilesInDirectoryWithTwelveThousandFiles, dirPath, PrefixFileInDirectoryWithTwelveThousandFiles, t)

	//operations.CreateDirectoryWithNFiles(NumberOfFilesInDirectoryWithTwelveThousandFiles, dirPath, PrefixFileInDirectoryWithTwelveThousandFiles, t)

	objs, err := os.ReadDir(dirPath)
	if err != nil {
		t.Errorf("Error in listing directory.")
	}

	// number of objs - 12000
	if len(objs) != NumberOfFilesInDirectoryWithTwelveThousandFiles {
		t.Errorf("Listed incorrect number of files from directory: %v, expected 12000", len(objs))
	}

	// Checking if all the object is File type.
	for i := 0; i < len(objs); i++ {
		if objs[i].IsDir() {
			t.Errorf("Listes object is incorrect.")
		}
	}

	for i := 0; i < len(objs); i++ {
		checkIfObjNameIsCorrect(objs[i].Name(), PrefixFileInDirectoryWithTwelveThousandFiles, NumberOfFilesInDirectoryWithTwelveThousandFiles, t)
	}
}

// Test with a bucket with twelve thousand files and hundred explicit directories.
func TestDirectoryWithTwelveThousandFilesAndHundredExplicitDir(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFiles)
	// Check if directory exist in previous test.
	_, err := os.Stat(dirPath)
	if err != nil {
		operations.CreateDirectoryWithNFiles(NumberOfFilesInDirectoryWithTwelveThousandFiles, dirPath, PrefixFileInDirectoryWithTwelveThousandFiles, t)
	}

	// Create hundred explicit directories.
	for i := 1; i <= 100; i++ {
		subDirPath := path.Join(dirPath, PrefixExplicitDirInLargeDirListTest+strconv.Itoa(i))
		// Create 100 Explicit directory.
		operations.CreateDirectoryWithNFiles(0, subDirPath, "", t)
	}

	objs, err := os.ReadDir(dirPath)
	if err != nil {
		t.Errorf("Error in listing directory.")
	}

	var numberOfFiles = 0
	var numberOfDirs = 0

	// Checking if correct objects present in bucket.
	for i := 0; i < len(objs); i++ {
		if !objs[i].IsDir() {
			numberOfFiles++
			// Checking if Prefix1 to Prefix12000 present in the bucket
			checkIfObjNameIsCorrect(objs[i].Name(), PrefixFileInDirectoryWithTwelveThousandFiles, NumberOfFilesInDirectoryWithTwelveThousandFiles, t)
		}

		if objs[i].IsDir() {
			numberOfDirs++
			// Checking if Prefix1 to Prefix100 present in the bucket
			checkIfObjNameIsCorrect(objs[i].Name(), PrefixExplicitDirInLargeDirListTest, NumberOfExplicitDirsInDirectoryWithTwelveThousandFiles, t)
		}
	}

	// number of explicit dirs = 100
	if numberOfDirs != NumberOfExplicitDirsInDirectoryWithTwelveThousandFiles {
		t.Errorf("Listed incorrect number of directories from directory: %v, expected 100", numberOfDirs)
	}
	// number of files = 12000
	if numberOfFiles != NumberOfFilesInDirectoryWithTwelveThousandFiles {
		t.Errorf("Listed incorrect number of files from directory: %v, expected 12000", numberOfFiles)
	}
}

// Test with a bucket with twelve thousand files, hundred explicit directories, and hundred implicit directories.
func TestDirectoryWithTwelveThousandFilesAndHundredExplicitDirAndHundredImplicitDir(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwelveThousandFiles)
	// Check if directory exist in previous test.
	_, err := os.Stat(dirPath)
	if err != nil {
		operations.CreateDirectoryWithNFiles(NumberOfFilesInDirectoryWithTwelveThousandFiles, dirPath, PrefixFileInDirectoryWithTwelveThousandFiles, t)
	}

	// Create hundred explicit directories.
	for i := 1; i <= 100; i++ {
		subDirPath := path.Join(dirPath, PrefixExplicitDirInLargeDirListTest+strconv.Itoa(i))

		// Check if directory exist in previous test.
		_, err := os.Stat(subDirPath)
		if err != nil {
			operations.CreateDirectoryWithNFiles(0, subDirPath, "", t)
		}
	}

	subDirPath := path.Join(setup.TestBucket(), DirectoryWithTwelveThousandFiles)
	setup.RunScriptForTestData("testdata/create_implicit_dir.sh", subDirPath)

	objs, err := os.ReadDir(dirPath)
	if err != nil {
		t.Errorf("Error in listing directory.")
	}

	var numberOfFiles = 0
	var numberOfDirs = 0

	// Checking if correct objects present in bucket.
	for i := 0; i < len(objs); i++ {
		if !objs[i].IsDir() {
			numberOfFiles++

			// Checking if Prefix1 to Prefix12000 present in the bucket
			checkIfObjNameIsCorrect(objs[i].Name(), PrefixFileInDirectoryWithTwelveThousandFiles, NumberOfFilesInDirectoryWithTwelveThousandFiles, t)
		}

		if objs[i].IsDir() {
			numberOfDirs++

			if strings.Contains(objs[i].Name(), PrefixExplicitDirInLargeDirListTest) {
				// Checking if explicitDir1 to explicitDir100 present in the bucket.
				checkIfObjNameIsCorrect(objs[i].Name(), PrefixExplicitDirInLargeDirListTest, NumberOfExplicitDirsInDirectoryWithTwelveThousandFiles, t)
			} else {
				// Checking if implicitDir1 to implicitDir100 present in the bucket.
				checkIfObjNameIsCorrect(objs[i].Name(), PrefixImplicitDirInLargeDirListTest, NumberOfImplicitDirsInDirectoryWithTwelveThousandFiles, t)
			}
		}
	}

	// number of dirs = 200(Number of implicit + Number of explicit directories)
	if numberOfDirs != NumberOfImplicitDirsInDirectoryWithTwelveThousandFiles+NumberOfExplicitDirsInDirectoryWithTwelveThousandFiles {
		t.Errorf("Listed incorrect number of directories from directory: %v, expected 200", numberOfDirs)
	}
	// number of files = 12000
	if numberOfFiles != NumberOfFilesInDirectoryWithTwelveThousandFiles {
		t.Errorf("Listed incorrect number of files from directory: %v, expected 12000", numberOfFiles)
	}

	// Clean the bucket for readonly testing.
	deleteTwelveThousandFiles(12000, dirPath, PrefixFileInDirectoryWithTwelveThousandFiles, t)
}
