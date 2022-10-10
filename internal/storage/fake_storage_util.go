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

type FakeStorageServer interface {
	GetTestFakeStorageObject() fakestorage.Object

	CreateFakeStorageServer(objects []fakestorage.Object) (*fakestorage.Server, error)

	CreateStorageHandle() (storageHandleObj *Storageclient)

	ShutDown(fakeStorageServer *fakestorage.Server)

	ReturnFakeStorageServer() *fakestorage.Server
}

type FakeStorage struct {
	FakeStorageServer *fakestorage.Server
}

func (f *FakeStorage) GetTestFakeStorageObject() fakestorage.Object {
	return fakestorage.Object{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: TestBucketName,
			Name:       TestObjectName,
			Generation: TestObjectGeneration,
		},
		Content: []byte(ContentInTestObject),
	}
}

func (f *FakeStorage) CreateFakeStorageServer(objects []fakestorage.Object) (*fakestorage.Server, error) {
	return fakestorage.NewServerWithOptions(fakestorage.Options{
		InitialObjects: objects,
		Host:           host,
		Port:           port,
	})
}

func (f *FakeStorage) ReturnFakeStorageServer() (*fakestorage.Server, error) {
	var err error
	if f.FakeStorageServer == nil {
		f.FakeStorageServer, err = f.CreateFakeStorageServer([]fakestorage.Object{f.GetTestFakeStorageObject()})
	}
	return f.FakeStorageServer, err
}

func (f *FakeStorage) CreateStorageHandle() (storageHandleObj *Storageclient) {
	storageHandleObj = &Storageclient{Client: f.FakeStorageServer.Client()}
	return
}

func (f *FakeStorage) ShutDown() {
	f.FakeStorageServer.Stop()
}

func CreateObject(server *fakestorage.Server, object fakestorage.Object) {
	server.CreateObject(object)
}
