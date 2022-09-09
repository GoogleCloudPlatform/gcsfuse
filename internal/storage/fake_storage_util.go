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

package storage

import (
	"github.com/fsouza/fake-gcs-server/fakestorage"
)

const port uint16 = 8081
const host string = "127.0.0.1"

const TestBucketName string = "gcsfuse-default-bucket"
const TestObjectName string = "gcsfuse/default.txt"
const ContentInTestObject string = "Hello GCSFuse!!!"
const TestObjectGeneration int64 = 780

func GetTestFakeStorageObject() fakestorage.Object {
	return fakestorage.Object{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: TestBucketName,
			Name:       TestObjectName,
			Generation: TestObjectGeneration,
		},
		Content: []byte(ContentInTestObject),
	}
}

func CreateFakeStorageServer(objects []fakestorage.Object) (*fakestorage.Server, error) {
	return fakestorage.NewServerWithOptions(fakestorage.Options{
		InitialObjects: objects,
		Host:           host,
		Port:           port,
	})
}

func CreateObject(server *fakestorage.Server, object fakestorage.Object) {
	server.CreateObject(object)
}
