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

package gcs

import (
	"crypto/md5"
	"time"

	storagev1 "google.golang.org/api/storage/v1"
)

const ContentEncodingGzip = "gzip"

// Object is a record representing a particular generation of a particular
// object name in GCS.
//
// See here for more information about its fields:
//
//	https://cloud.google.com/storage/docs/json_api/v1/objects#resource
type Object struct {
	Name            string
	ContentType     string
	ContentLanguage string
	CacheControl    string
	Owner           string
	Size            uint64
	ContentEncoding string
	MD5             *[md5.Size]byte // Missing for composite objects
	CRC32C          *uint32         //Missing for CMEK buckets
	MediaLink       string
	Metadata        map[string]string
	Generation      int64
	MetaGeneration  int64
	StorageClass    string
	Deleted         time.Time
	Updated         time.Time

	// As of 2015-06-03, the official GCS documentation for this property
	// (https://tinyurl.com/2zjza2cu) says this:
	//
	//     Newly uploaded objects have a component count of 1, and composing a
	//     sequence of objects creates an object whose component count is equal
	//     to the sum of component counts in the sequence.
	//
	// However, in Google-internal bug 21572928 it was clarified that this
	// doesn't match the actual implementation, which can be documented as:
	//
	//     Newly uploaded objects do not have a component count. Composing a
	//     sequence of objects creates an object whose component count is equal
	//     to the sum of the component counts of the objects in the sequence,
	//     where objects that do not have a component count are treated as having
	//     a component count of 1.
	//
	// This is a much less elegant and convenient rule, so this package emulates
	// the officially documented behavior above. That is, it synthesizes a
	// component count of 1 for objects that do not have a component count.
	ComponentCount int64

	ContentDisposition string
	CustomTime         string
	EventBasedHold     bool
	Acl                []*storagev1.ObjectAccessControl
}

// MinObject is a record representing subset of properties of a particular
// generation object in GCS.
//
// See here for more information about its fields:
//
//	https://cloud.google.com/storage/docs/json_api/v1/objects#resource
type MinObject struct {
	Name            string
	Size            uint64
	generation      int64
	MetaGeneration  int64
	Updated         time.Time
	Metadata        map[string]string
	ContentEncoding string
	CRC32C          *uint32 // Missing for CMEK buckets
}

// ExtendedObjectAttributes contains the missing attributes of Object which are not present in MinObject.
type ExtendedObjectAttributes struct {
	ContentType        string
	ContentLanguage    string
	CacheControl       string
	Owner              string
	MD5                *[md5.Size]byte // Missing for composite objects
	MediaLink          string
	StorageClass       string
	Deleted            time.Time
	ComponentCount     int64
	ContentDisposition string
	CustomTime         string
	EventBasedHold     bool
	Acl                []*storagev1.ObjectAccessControl
}

func NewMinObject(name string, size uint64, generation int64,
	metaGeneration int64, updated time.Time, metadata map[string]string,
	contentEncoding string, crc32c *uint32) *MinObject {
	// Make a copy of crc32c to not reference callers crc32c.
	if crc32c != nil {
		copyCrc32c := *crc32c
		crc32c = &copyCrc32c
	}
	return &MinObject{
		Name:            name,
		Size:            size,
		generation:      generation,
		MetaGeneration:  metaGeneration,
		Updated:         updated,
		Metadata:        metadata,
		ContentEncoding: contentEncoding,
		CRC32C:          crc32c,
	}
}

func (mo MinObject) HasContentEncodingGzip() bool {
	return mo.ContentEncoding == ContentEncodingGzip
}

func (m *MinObject) Generation() int64 {
	return m.generation
}

func SetGenerationForTesting(m *MinObject, generation int64) {
	m.generation = generation
}
