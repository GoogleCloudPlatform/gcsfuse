// Copyright 2015 Google Inc. All Rights Reserved.
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

// Object is a record representing a particular generation of a particular
// object name in GCS.
//
// See here for more information about its fields:
//
//     https://cloud.google.com/storage/docs/json_api/v1/objects#resource
//
type Object struct {
	Name            string
	ContentType     string
	ContentLanguage string
	CacheControl    string
	Owner           string
	Size            uint64
	ContentEncoding string
	MD5             *[md5.Size]byte // Missing for composite objects
	CRC32C          *uint32 //Missing for CMEK buckets
	MediaLink       string
	Metadata        map[string]string
	Generation      int64
	MetaGeneration  int64
	StorageClass    string
	Deleted         time.Time
	Updated         time.Time

	// NOTE(jacobsa): As of 2015-06-03, the official GCS documentation for this
	// property (https://goo.gl/GwD5Dq) says this:
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
