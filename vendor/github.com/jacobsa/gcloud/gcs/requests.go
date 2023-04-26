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
	"fmt"
	"io"

	storagev1 "google.golang.org/api/storage/v1"
)

// A request to create an object, accepted by Bucket.CreateObject.
type CreateObjectRequest struct {
	// The name with which to create the object. This field must be set.
	//
	// Object names must:
	//
	// *  be non-empty.
	// *  be no longer than 1024 bytes.
	// *  be valid UTF-8.
	// *  not contain the code point U+000A (line feed).
	// *  not contain the code point U+000D (carriage return).
	//
	// See here for authoritative documentation:
	//     https://cloud.google.com/storage/docs/bucket-naming#objectnames
	Name string

	// Optional information with which to create the object. See here for more
	// information:
	//
	//     https://cloud.google.com/storage/docs/json_api/v1/objects#resource
	//
	ContentType        string
	ContentLanguage    string
	ContentEncoding    string
	CacheControl       string
	Metadata           map[string]string
	ContentDisposition string
	CustomTime         string
	EventBasedHold     bool
	StorageClass       string
	Acl                []*storagev1.ObjectAccessControl

	// A reader from which to obtain the contents of the object. Must be non-nil.
	Contents io.Reader

	// If non-nil, the object will not be created if the checksum of the received
	// contents does not match the supplied value.
	CRC32C *uint32

	// If non-nil, the object will not be created if the MD5 sum of the received
	// contents does not match the supplied value.
	MD5 *[md5.Size]byte

	// If non-nil, the object will be created/overwritten only if the current
	// generation for the object name is equal to the given value. Zero means the
	// object does not exist.
	GenerationPrecondition *int64

	// If non-nil, the object will be created/overwritten only if the current
	// meta-generation for the object name is equal to the given value. This is
	// only meaningful in conjunction with GenerationPrecondition.
	MetaGenerationPrecondition *int64
}

// A request to copy an object to a new name, preserving all metadata.
type CopyObjectRequest struct {
	SrcName string
	DstName string

	// The generation of the source object to copy, or zero for the latest
	// generation.
	SrcGeneration int64

	// If non-nil, the destination object will be created/overwritten only if the
	// current meta-generation for the source object is equal to the given value.
	//
	// This is probably only meaningful in conjunction with SrcGeneration.
	SrcMetaGenerationPrecondition *int64

	// Destination object will be overwritten only if the current
	// generation is equal to the given value. Zero means the object does not
	// exist.
	DstGenerationPrecondition *int64
}

// MaxSourcesPerComposeRequest is the maximum number of sources that a
// ComposeObjectsRequest may contain.
//
// Cf. https://cloud.google.com/storage/docs/composite-objects#_Compose
const MaxSourcesPerComposeRequest = 32

// MaxComponentCount is the maximum number of components that a composite
// object may have. The sum of the component counts of the sources in a
// ComposeObjectsRequest must be no more than this value.
//
// Cf. https://cloud.google.com/storage/docs/composite-objects#_Count
const MaxComponentCount = 1024

// A request to compose one or more objects into a single composite object.
type ComposeObjectsRequest struct {
	// The name of the destination composite object.
	DstName string

	// If non-nil, the destination object will be created/overwritten only if the
	// current generation for its name is equal to the given value. Zero means
	// the object does not exist.
	DstGenerationPrecondition *int64

	// If non-nil, the destination object will be created/overwritten only if the
	// current meta-generation for its name is equal to the given value.
	//
	// This is only meaningful if DstGenerationPrecondition is also specified.
	DstMetaGenerationPrecondition *int64

	// The source objects from which to compose. This must be non-empty.
	//
	// Make sure to see the notes on MaxSourcesPerComposeRequest and
	// MaxComponentCount.
	Sources []ComposeSource

	// Optional information with which to create the object. See here for more
	// information:
	//
	//     https://cloud.google.com/storage/docs/json_api/v1/objects#resource
	//
	ContentType        string
	Metadata           map[string]string
	ContentLanguage    string
	ContentEncoding    string
	CacheControl       string
	ContentDisposition string
	CustomTime         string
	EventBasedHold     bool
	StorageClass       string
	Acl                []*storagev1.ObjectAccessControl
}

type ComposeSource struct {
	// The name of the source object.
	Name string

	// The generation of the source object to compose from. Zero means the latest
	// generation.
	Generation int64
}

// ByteRange is a [start, limit) range of bytes within an object.
//
// Its semantics are as follows:
//
//  *  If Limit is less than or equal to Start, the range is treated as empty.
//
//  *  The effective range is [start, limit) intersected with [0, L), where L
//     is the length of the object.
//
//     For example, a read for [L-1, L+10) returns the last byte of the object,
//     and [L+2, L+10) is legal but returns nothing.
//
type ByteRange struct {
	Start uint64
	Limit uint64
}

func (br ByteRange) String() string {
	return fmt.Sprintf("[%d, %d)", br.Start, br.Limit)
}

// A request to read the contents of an object at a particular generation.
type ReadObjectRequest struct {
	// The name of the object to read.
	Name string

	// The generation of the object to read. Zero means the latest generation.
	Generation int64

	// If present, limit the contents returned to a range within the object.
	Range *ByteRange
}

type StatObjectRequest struct {
	// The name of the object in question.
	Name string

	// Relevant only when fast_stat_bucket is used. This field controls whether
	// to fetch from gcs or from cache.
	ForceFetchFromGcs bool
}

type Projection int64

const (
	Full Projection = iota
	NoAcl
)

// Returning the string values based on the values accepted by projection param.
// https://cloud.google.com/storage/docs/json_api/v1/objects/list#parameters
func (p Projection) String() string {
	switch p {
	case Full:
		return "full"
	case NoAcl:
		return "noAcl"
	}
	return "full"
}

type ListObjectsRequest struct {
	// List only objects whose names begin with this prefix.
	Prefix string

	// Collapse results based on a delimiter.
	//
	// If non-empty, enable the following behavior. For each run of one or more
	// objects whose names are of the form:
	//
	//     <Prefix><S><Delimiter><...>
	//
	// where <S> is a string that doesn't itself contain Delimiter and <...> is
	// anything, return a single Collaped entry in the listing consisting of
	//
	//     <Prefix><S><Delimiter>
	//
	// instead of one Object record per object. If a collapsed entry consists of
	// a large number of objects, this may be more efficient.
	Delimiter string

	// Only applicable when Delimiter is set nonempty. Default to false.
	//
	// The objects in the result listing would contain those objects that end
	// with the first delimiter if this is true. Otherwise, those objects are
	// not included.
	//
	// Example:
	//  Assume there is an object "foo/bar/".
	//  1. Prefix: "foo/", Delimiter: "/", IncludeTrailingDelimiter: true
	//     -> "foo/bar/" exists in both listing.CollapsedRuns and
	//     listing.Objects.
	//  2. Prefix: "foo/", Delimiter: "/", IncludeTrailingDelimiter: false
	//     -> "foo/bar/" exists in only listing.CollapsedRuns but not
	//     listing.Objects.
	IncludeTrailingDelimiter bool

	// Used to continue a listing where a previous one left off. See
	// Listing.ContinuationToken for more information.
	ContinuationToken string

	// The maximum number of objects and collapsed runs to return. Fewer than
	// this number may actually be returned. If this is zero, a sensible default
	// is used.
	MaxResults int

	// Set of properties to return. Acceptable values- full & noAcl.
	//    1. full  - returns all properties
	//    2. noAcl - omit owner, acl properties
	//
	// Currently projection value is hardcoded to full. To keep it aligned with
	// the current flow, default value will be full and callers can override it
	// using this param.
	ProjectionVal Projection
}

// Listing contains a set of objects and delimter-based collapsed runs returned
// by a call to ListObjects. See also ListObjectsRequest.
type Listing struct {
	// Records for objects matching the listing criteria.
	//
	// Guaranteed to be strictly increasing under a lexicographical comparison on
	// (name, generation) pairs.
	Objects []*Object

	// Collapsed entries for runs of names sharing a prefix followed by a
	// delimiter. See notes on ListObjectsRequest.Delimiter.
	//
	// Guaranteed to be strictly increasing.
	CollapsedRuns []string

	// A continuation token, for fetching more results.
	//
	// If non-empty, this listing does not represent the full set of matching
	// objects in the bucket. Call ListObjects again with the request's
	// ContinuationToken field set to this value to continue where you left off.
	//
	// Guarantees, for replies R1 and R2, with R2 continuing from R1:
	//
	//  *  All of R1's object names are strictly less than all object names and
	//     collapsed runs in R2.
	//
	//  *  All of R1's collapsed runs are strictly less than all object names and
	//     prefixes in R2.
	//
	// (Cf. Google-internal bug 19286144)
	//
	// Note that there is no guarantee of atomicity of listings. Objects written
	// and deleted concurrently with a single or multiple listing requests may or
	// may not be returned.
	ContinuationToken string
}

// A request to update the metadata of an object, accepted by
// Bucket.UpdateObject.
type UpdateObjectRequest struct {
	// The name of the object to update. Must be specified.
	Name string

	// The generation of the object to update. Zero means the latest generation.
	Generation int64

	// If non-nil, the request will fail without effect if there is an object
	// with the given name (and optionally generation), and its meta-generation
	// is not equal to this value.
	MetaGenerationPrecondition *int64

	// String fields in the object to update (or not). The semantics are as
	// follows, for a given field F:
	//
	//  *  If F is set to nil, the corresponding GCS object field is untouched.
	//
	//  *  If *F is the empty string, then the corresponding GCS object field is
	//     removed.
	//
	//  *  Otherwise, the corresponding GCS object field is set to *F.
	//
	//  *  There is no facility for setting a GCS object field to the empty
	//     string, since many of the fields do not actually allow that as a legal
	//     value.
	//
	// Note that the GCS object's content type field cannot be removed.
	ContentType     *string
	ContentEncoding *string
	ContentLanguage *string
	CacheControl    *string

	// User-provided metadata updates. Keys that are not mentioned are untouched.
	// Keys whose values are nil are deleted, and others are updated to the
	// supplied string. There is no facility for completely removing user
	// metadata.
	Metadata map[string]*string
}

// A request to delete an object by name. Non-existence is not treated as an
// error.
type DeleteObjectRequest struct {
	// The name of the object to delete. Must be specified.
	Name string

	// The generation of the object to delete. Zero means the latest generation.
	Generation int64

	// If non-nil, the request will fail without effect if there is an object
	// with the given name (and optionally generation), and its meta-generation
	// is not equal to this value.
	MetaGenerationPrecondition *int64
}
