//Copyright 2023 Google LLC
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

package all_mounting

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/persistent_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

// MountingType represents the different types of GCSFuse mounting.
type MountingType int

const (
	// StaticMounting represents mounting a single bucket to a directory.
	StaticMounting MountingType = iota
	// DynamicMounting represents mounting all accessible buckets under a directory.
	DynamicMounting
	// PersistentMounting represents mounting defined in /etc/fstab.
	PersistentMounting
	// OnlyDirMounting represents mounting only a specific directory within a bucket.
	OnlyDirMounting
)

const (
	onlyDirMountTestPrefix = "onlyDirMountTest-"
)

type TestMountConfiguration struct {
	basePackageTestDir                 string
	namedTestDir                       string
	flags                              []string
	mountType                          MountingType
	logFile                            string
	rootMntDir                         string
	onlyDirExistsOnBucket              bool   // scenario when onlyDir exists on testBucket.
	onlyDir                            string // represents onlyDir if it's OnlyDirMounting.
	useCreatedBucketForDynamicMounting bool   // Use createdBucket for dynamic Mounting.
	createdBucket                      string
	dynmaincMntDir                     string // MntDir for dynamic mounting.
}

func (t *TestMountConfiguration) LogFile() string {
	if t.logFile == "" {
		log.Println("Log file path is not set up yet. Ensure Mount() has been invoked successfully before calling LogFile().")
		os.Exit(1)
	}
	return t.logFile
}

func (t *TestMountConfiguration) MntDir() string {
	if t.dynmaincMntDir != "" {
		return t.dynmaincMntDir
	}
	if t.rootMntDir == "" {
		log.Println("MntDir is not set up yet. Ensure Mount() has been invoked successfully before calling MntDir().")
		os.Exit(1)
	}
	return t.rootMntDir
}

func (t *TestMountConfiguration) MountType() MountingType {
	return t.mountType
}

func (t *TestMountConfiguration) Mount(tb testing.TB, testName string, storageClient *storage.Client) error {
	t.namedTestDir = path.Join(t.basePackageTestDir, testName)
	err := os.Mkdir(t.namedTestDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create test named directory '%s': %w", t.namedTestDir, err)
	}
	rootMntDir := path.Join(t.namedTestDir, "mnt")
	err = os.Mkdir(rootMntDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create 'mnt' directory inside '%s': %w", t.namedTestDir, err)
	}

	t.logFile = path.Join(t.namedTestDir, "gcsfuse.log")

	switch t.mountType {
	case StaticMounting:
		err = static_mounting.MountGcsfuseWithStaticMountingMntDirAndLogFile(t.flags, rootMntDir, t.logFile)
	case PersistentMounting:
		err = persistent_mounting.MountGcsfuseWithPersistentMountingMntDirLogFile(t.flags, rootMntDir, t.logFile)
	case DynamicMounting:
		ctx := context.Background()
		if t.useCreatedBucketForDynamicMounting {
			t.createdBucket = dynamic_mounting.CreateTestBucketForDynamicMounting(ctx, storageClient)
		}
		err = dynamic_mounting.MountGcsfuseWithDynamicMountingMntDirLogFile(t.flags, rootMntDir, t.logFile)
	case OnlyDirMounting:
		ctx := context.Background()
		t.onlyDir = onlyDirMountTestPrefix + setup.GenerateRandomString(5)
		if t.onlyDirExistsOnBucket {
			_ = client.SetupTestDirectoryMntDirOnlyDir(ctx, storageClient, t.onlyDir, rootMntDir, t.onlyDir)
		} else {
			err = client.DeleteAllObjectsWithPrefix(ctx, storageClient, t.onlyDir)
			if err != nil {
				return fmt.Errorf("failed to clean up Objects with prefix %s for only directory mounting: %w", t.onlyDir, err)
			}
		}
		err = only_dir_mounting.MountGcsfuseWithOnlyDirMntDirLogFile(t.flags, rootMntDir, t.logFile, t.onlyDir)
	default:
		return fmt.Errorf("unknown mount type: %v", t.mountType)
	}
	if err != nil {
		return fmt.Errorf("failed to mount GCSFuse for mountType: %v, err: %w", t.mountType, err)
	}
	t.rootMntDir = rootMntDir
	if t.mountType == DynamicMounting {
		if t.useCreatedBucketForDynamicMounting {
			t.dynmaincMntDir = path.Join(rootMntDir, t.createdBucket)
		} else {
			t.dynmaincMntDir = path.Join(rootMntDir, setup.TestBucket())
		}
	}
	return nil
}

func (t *TestMountConfiguration) Unmount() error {
	fusermount, err := exec.LookPath("fusermount")
	if err != nil {
		return fmt.Errorf("cannot find fusermount: %w", err)
	}
	cmd := exec.Command(fusermount, "-uz", t.rootMntDir)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("fusermount error: %w", err)
	}
	return nil

}

func GenerateTestMountConfigurations(mountTypes []MountingType, flagsSet [][]string, baseTestDir string) []TestMountConfiguration {
	var testMountConfigurations []TestMountConfiguration
	for _, mountType := range mountTypes {
		for _, flags := range flagsSet {
			testMountConfiguration := TestMountConfiguration{
				mountType:          mountType,
				flags:              flags,
				basePackageTestDir: baseTestDir,
			}
			if mountType == OnlyDirMounting {
				dup := testMountConfiguration
				dup.onlyDirExistsOnBucket = true
				testMountConfigurations = append(testMountConfigurations, dup)
			}
			if mountType == DynamicMounting {
				dup := testMountConfiguration
				dup.useCreatedBucketForDynamicMounting = true
				testMountConfigurations = append(testMountConfigurations, dup)
			}
			testMountConfigurations = append(testMountConfigurations, testMountConfiguration)
		}
	}
	return testMountConfigurations
}

func UnmountAll(mountConfiguration []TestMountConfiguration, storageClient *storage.Client) error {
	cnt := 0
	for _, testMountConfiguration := range mountConfiguration {
		if testMountConfiguration.rootMntDir != "" {
			err := testMountConfiguration.Unmount()
			if err != nil {
				cnt++
				log.Printf("Unable to unmount mntDir: %s, err: %v", testMountConfiguration.rootMntDir, err)
			} else {
				log.Printf("Successfully unmounted mntDir: %s", testMountConfiguration.rootMntDir)
			}
			if testMountConfiguration.mountType == DynamicMounting && testMountConfiguration.useCreatedBucketForDynamicMounting {
				err := client.DeleteBucket(context.Background(), storageClient, testMountConfiguration.createdBucket)
				if err != nil {
					log.Printf("Unable to delete bucket: %s, err: %v", testMountConfiguration.createdBucket, err)
				} else {
					log.Printf("Successfully deleted bucket: %s", testMountConfiguration.createdBucket)
				}
			}
		}
	}
	if cnt > 0 {
		return fmt.Errorf("failed to unmount %d configurations", cnt)
	}
	return nil
}
