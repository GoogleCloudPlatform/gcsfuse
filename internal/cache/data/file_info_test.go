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

package data

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const TestDataFileSize uint64 = 23
const TestTimeInEpoch int64 = 1654041600
const TestBucketName = "test_bucket"
const TestObjectName = "test/a.txt"
const TestGeneration = 12345678
const ExpectedFileInfoKey = "test_bucket1654041600test/a.txt"

func getTestFileInfoKey() FileInfoKey {
	key, err := NewFileInfoKey(TestBucketName, TestTimeInEpoch, TestObjectName)
	if err != nil {
		panic(err)
	}
	return key
}

func TestKeyMethod(t *testing.T) {
	fik := getTestFileInfoKey()

	key, err := fik.Key()

	assert.NoError(t, err)
	unixCreationTimeString := fmt.Sprintf("%d", fik.BucketCreationTime())
	assert.Equal(t, fik.BucketName()+"\x00"+unixCreationTimeString+"\x00"+fik.ObjectName(), key)
}

func TestKeyMethodWithEmptyBucketName(t *testing.T) {
	fik := getTestFileInfoKey()
	fik.cachedKey = ""
	fik.bucketName = ""

	key, err := fik.Key()

	assert.Equal(t, InvalidKeyAttributes, err.Error())
	assert.Equal(t, "", key)
}

func TestKeyMethodWithEmptyObjectName(t *testing.T) {
	fik := getTestFileInfoKey()
	fik.cachedKey = ""
	fik.objectName = ""

	key, err := fik.Key()

	assert.Equal(t, InvalidKeyAttributes, err.Error())
	assert.Equal(t, "", key)
}

func TestContentSizeMethod(t *testing.T) {
	fileContentSize := uint64(23)
	blockSize := uint64(4096)
	fi := NewFileInfo(getTestFileInfoKey(), TestGeneration, fileContentSize, 0, false, nil, blockSize)

	contentSize := fi.ContentSize()

	// ContentSize() always returns content-size.
	assert.Equal(t, fileContentSize, contentSize)
}

func TestSizeMethod(t *testing.T) {
	fileContentSize := uint64(23)
	blockSize := uint64(4096)
	fi := NewFileInfo(getTestFileInfoKey(), TestGeneration, fileContentSize, 0, false, nil, blockSize)

	size := fi.Size()

	// Size() returns size on disk, which is always multiples of block-size. So,
	// if content-size < block-size, then Size() return block-size.
	assert.Equal(t, blockSize, size)
}

func TestFileInfoKeyEquivalence(t *testing.T) {
	// Seed random generator.
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Test a wide range of random bucket names, object names, and creation times.
	for i := 0; i < 1000; i++ {
		bucketName := fmt.Sprintf("bucket-%d", r.Intn(100000))
		objectName := fmt.Sprintf("dir1/dir2/dir3/object-%d.txt", r.Intn(100000))
		creationTime := int64(r.Intn(2000000000))

		fikCurrent := FileInfoKey{
			bucketName:         bucketName,
			objectName:         objectName,
			bucketCreationTime: creationTime,
		}

		keyCurrent, errCurrent := fikCurrent.Key()
		if errCurrent != nil {
			t.Fatalf("Unexpected error from current implementation: %v", errCurrent)
		}

		fikOptimized, errOpt := NewFileInfoKey(bucketName, creationTime, objectName)
		if errOpt != nil {
			t.Fatalf("Unexpected error from NewFileInfoKey: %v", errOpt)
		}

		keyOptimized, errKeyOpt := fikOptimized.Key()
		if errKeyOpt != nil {
			t.Fatalf("Unexpected error from optimized Key(): %v", errKeyOpt)
		}

		if keyCurrent != keyOptimized {
			t.Errorf("Mismatch in generated keys:\ncurrent:   %q\noptimized: %q", keyCurrent, keyOptimized)
		}
	}
}

func TestFileInfoKeyEmptyAttributes(t *testing.T) {
	// 1. Current implementation behavior on empty attributes.
	fikCurrentEmptyBucket := FileInfoKey{
		bucketName:         "",
		objectName:         "object",
		bucketCreationTime: 1654041600,
	}
	_, err1 := fikCurrentEmptyBucket.Key()
	if err1 == nil {
		t.Error("Expected error for empty bucket name in current implementation, got nil")
	}

	fikCurrentEmptyObject := FileInfoKey{
		bucketName:         "bucket",
		objectName:         "",
		bucketCreationTime: 1654041600,
	}
	_, err2 := fikCurrentEmptyObject.Key()
	if err2 == nil {
		t.Error("Expected error for empty object name in current implementation, got nil")
	}

	// 2. Constructor behavior on empty attributes.
	_, errOpt1 := NewFileInfoKey("", 1654041600, "object")
	if errOpt1 == nil {
		t.Error("Expected error from NewFileInfoKey for empty bucket, got nil")
	}

	_, errOpt2 := NewFileInfoKey("bucket", 1654041600, "")
	if errOpt2 == nil {
		t.Error("Expected error from NewFileInfoKey for empty object, got nil")
	}

	// 3. Literal initialization with valid attributes (cachedKey is empty) -> Lazy Fallback succeeds.
	fikOptimizedUninitialized := FileInfoKey{
		bucketName:         "bucket",
		objectName:         "object",
		bucketCreationTime: 1654041600,
	}
	keyOpt, errOpt3 := fikOptimizedUninitialized.Key()
	if errOpt3 != nil {
		t.Errorf("Unexpected error from lazy fallback: %v", errOpt3)
	}
	expectedKey, _ := GetFileInfoKeyName("object", 1654041600, "bucket")
	if keyOpt != expectedKey {
		t.Errorf("Expected lazy fallback to generate key %q, got %q", expectedKey, keyOpt)
	}

	// 4. Literal initialization with invalid attributes (cachedKey is empty) -> Lazy Fallback fails and returns error.
	fikOptimizedInvalid := FileInfoKey{
		bucketName:         "",
		objectName:         "object",
		bucketCreationTime: 1654041600,
	}
	_, errOpt4 := fikOptimizedInvalid.Key()
	if errOpt4 == nil {
		t.Error("Expected error from lazy fallback for invalid attributes, got nil")
	}
}

func TestFileInfoKeyStructComparison(t *testing.T) {
	fik1, err := NewFileInfoKey("bucket", 1654041600, "object")
	assert.NoError(t, err)

	fik2, err := NewFileInfoKey("bucket", 1654041600, "object")
	assert.NoError(t, err)

	// Since both are constructed via NewFileInfoKey, their fields (including private cachedKey) are equal.
	assert.Equal(t, fik1, fik2)

	m := make(map[FileInfoKey]bool)
	m[fik2] = true

	// Lookup using another constructed key will succeed!
	assert.True(t, m[fik1])
}

func TestFileInfoKey_Collision(t *testing.T) {
	// Case 1: Bucket name is "name", creation time is 1700000000, object is "obj"
	fik1, err1 := NewFileInfoKey("name", 1700000000, "obj")
	assert.NoError(t, err1)
	key1, errKey1 := fik1.Key()
	assert.NoError(t, errKey1)

	// Case 2: Bucket name is "name17000", creation time is 0, object is "0000obj"
	fik2, err2 := NewFileInfoKey("name17000", 0, "0000obj")
	assert.NoError(t, err2)
	key2, errKey2 := fik2.Key()
	assert.NoError(t, errKey2)

	t.Logf("Key 1: %s", key1)
	t.Logf("Key 2: %s", key2)

	if key1 == key2 {
		t.Errorf("COLLISION DETECTED: Both distinct FileInfoKeys produced the same key string: %q", key1)
	}
}

func TestFileInfoKey_CollisionWithZeroCreationTime(t *testing.T) {
	// In production gcsfuse, bucket creation time is often passed as 0.
	// Let's verify collision under this constraint.
	
	// Case 1: Bucket "bucket", object "0object"
	fik1, err1 := NewFileInfoKey("bucket", 0, "0object")
	assert.NoError(t, err1)
	key1, errKey1 := fik1.Key()
	assert.NoError(t, errKey1)

	// Case 2: Bucket "bucket0", object "object"
	fik2, err2 := NewFileInfoKey("bucket0", 0, "object")
	assert.NoError(t, err2)
	key2, errKey2 := fik2.Key()
	assert.NoError(t, errKey2)

	t.Logf("Key 1: %s", key1)
	t.Logf("Key 2: %s", key2)

	if key1 == key2 {
		t.Errorf("COLLISION DETECTED (Zero creation time): Both distinct FileInfoKeys produced the same key string: %q", key1)
	}
}

func TestFileInfoKey_Adversarial(t *testing.T) {
	tests := []struct {
		name               string
		bucketName         string
		bucketCreationTime int64
		objectName         string
		expectError        bool
	}{
		{
			name:               "Standard valid inputs",
			bucketName:         "my-bucket",
			bucketCreationTime: 1654041600,
			objectName:         "photos/family.png",
			expectError:        false,
		},
		{
			name:               "Empty bucket name",
			bucketName:         "",
			bucketCreationTime: 1654041600,
			objectName:         "photos/family.png",
			expectError:        true,
		},
		{
			name:               "Empty object name",
			bucketName:         "my-bucket",
			bucketCreationTime: 1654041600,
			objectName:         "",
			expectError:        true,
		},
		{
			name:               "Negative bucket creation time",
			bucketName:         "my-bucket",
			bucketCreationTime: -3600, // 1 hour before epoch
			objectName:         "photos/family.png",
			expectError:        false,
		},
		{
			name:               "Zero bucket creation time",
			bucketName:         "my-bucket",
			bucketCreationTime: 0,
			objectName:         "photos/family.png",
			expectError:        false,
		},
		{
			name:               "Object name containing null bytes",
			bucketName:         "my-bucket",
			bucketCreationTime: 1654041600,
			objectName:         "photos\x00family\x00.png",
			expectError:        false,
		},
		{
			name:               "Object name containing special characters",
			bucketName:         "my-bucket",
			bucketCreationTime: 1654041600,
			objectName:         "photos/!@#$%^&*()_+{}|:\"<>?`-=[]\\;',./.png",
			expectError:        false,
		},
		{
			name:               "Object name containing newlines",
			bucketName:         "my-bucket",
			bucketCreationTime: 1654041600,
			objectName:         "photos/\nfamily\r\n.png",
			expectError:        false,
		},
		{
			name:               "Bucket name containing null bytes (malicious/invalid)",
			bucketName:         "my\x00bucket",
			bucketCreationTime: 1654041600,
			objectName:         "photos/family.png",
			expectError:        false, // Technically valid in struct construction, though invalid in GCS.
		},
	}

	generatedKeys := make(map[string]string) // keyString -> testCaseName

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fik, err := NewFileInfoKey(tc.bucketName, tc.bucketCreationTime, tc.objectName)
			if tc.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			keyStr, err := fik.Key()
			assert.NoError(t, err)
			assert.NotEmpty(t, keyStr)

			// Check for collisions with other test cases
			if existingName, exists := generatedKeys[keyStr]; exists {
				t.Errorf("COLLISION: case %q and case %q generated the identical key string %q", tc.name, existingName, keyStr)
			}
			generatedKeys[keyStr] = tc.name

			// Verify getters return exact inputs
			assert.Equal(t, tc.bucketName, fik.BucketName())
			assert.Equal(t, tc.bucketCreationTime, fik.BucketCreationTime())
			assert.Equal(t, tc.objectName, fik.ObjectName())
		})
	}
}

func TestFileInfoKey_PossibleCollision(t *testing.T) {
	// If bucketName contains null bytes, it could potentially collide with another key.
	// Let's see if this is possible.
	fik1, err1 := NewFileInfoKey("b\x00123", 0, "o")
	assert.NoError(t, err1)
	key1, errKey1 := fik1.Key()
	assert.NoError(t, errKey1)

	fik2, err2 := NewFileInfoKey("b", 123, "0\x00o")
	assert.NoError(t, err2)
	key2, errKey2 := fik2.Key()
	assert.NoError(t, errKey2)

	t.Logf("Key 1: %q", key1)
	t.Logf("Key 2: %q", key2)

	if key1 == key2 {
		t.Logf("Theoretical collision possible if bucketName contains null bytes: %q == %q", key1, key2)
	}
}



