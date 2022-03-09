// Copyright 2022 Google Inc. All Rights Reserved.
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

package contentcache_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/contentcache"
	"github.com/jacobsa/fuse/fsutil"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

const testTempDir = "/tmp"
const testGeneration = 10002022

func TestValidateGeneration(t *testing.T) {
	objectMetadata := contentcache.CacheFileObjectMetadata{
		CacheFileNameOnDisk: "foobar",
		BucketName:          "foo",
		ObjectName:          "baz",
		Generation:          testGeneration,
		MetaGeneration:      1,
	}
	cacheObject := contentcache.CacheObject{CacheFileObjectMetadata: &objectMetadata}
	ExpectTrue(cacheObject.ValidateGeneration(testGeneration))
}

func TestReadWriteMetadataCheckpointFile(t *testing.T) {
	mtimeClock := timeutil.RealClock()
	contentCache := contentcache.New(testTempDir, mtimeClock)
	f, err := fsutil.AnonymousFile(testTempDir)
	AssertEq(err, nil)
	objectMetadata := contentcache.CacheFileObjectMetadata{
		CacheFileNameOnDisk: f.Name(),
		BucketName:          "foo",
		ObjectName:          "baz",
		Generation:          testGeneration,
		MetaGeneration:      1,
	}
	metadataFileName, err := contentCache.WriteMetadataCheckpointFile(objectMetadata.ObjectName, &objectMetadata)
	AssertEq(err, nil)
	newObjectMetadata := contentcache.CacheFileObjectMetadata{}
	contents, err := ioutil.ReadFile(metadataFileName)
	AssertEq(err, nil)
	err = json.Unmarshal(contents, &newObjectMetadata)
	AssertEq(err, nil)
	// There is no struct equality support in ExpectEq
	ExpectEq(objectMetadata.BucketName, newObjectMetadata.BucketName)
	ExpectEq(objectMetadata.CacheFileNameOnDisk, newObjectMetadata.CacheFileNameOnDisk)
	ExpectEq(objectMetadata.Generation, newObjectMetadata.Generation)
	ExpectEq(objectMetadata.MetaGeneration, newObjectMetadata.MetaGeneration)
	ExpectEq(objectMetadata.ObjectName, newObjectMetadata.ObjectName)
	os.Remove(metadataFileName)
}
