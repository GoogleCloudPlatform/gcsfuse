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

package gcsfake

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/syncutil"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
)

var crc32cTable = crc32.MakeTable(crc32.Castagnoli)

// Equivalent to NewConn(clock).GetBucket(name).
func NewFakeBucket(clock timeutil.Clock, name string) gcs.Bucket {
	b := &bucket{clock: clock, name: name}
	b.mu = syncutil.NewInvariantMutex(b.checkInvariants)
	return b
}

////////////////////////////////////////////////////////////////////////
// Helper types
////////////////////////////////////////////////////////////////////////

type fakeObject struct {
	metadata gcs.Object
	data     []byte
}

// A slice of objects compared by name.
type fakeObjectSlice []fakeObject

func (s fakeObjectSlice) Len() int {
	return len(s)
}

func (s fakeObjectSlice) Less(i, j int) bool {
	return s[i].metadata.Name < s[j].metadata.Name
}

func (s fakeObjectSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Return the smallest i such that s[i].metadata.Name >= name, or len(s) if
// there is no such i.
func (s fakeObjectSlice) lowerBound(name string) int {
	pred := func(i int) bool {
		return s[i].metadata.Name >= name
	}

	return sort.Search(len(s), pred)
}

// Return the smallest i such that s[i].metadata.Name == name, or len(s) if
// there is no such i.
func (s fakeObjectSlice) find(name string) int {
	lb := s.lowerBound(name)
	if lb < len(s) && s[lb].metadata.Name == name {
		return lb
	}

	return len(s)
}

// Return the smallest string that is lexicographically larger than prefix and
// does not have prefix as a prefix. For the sole case where this is not
// possible (all strings consisting solely of 0xff bytes, including the empty
// string), return the empty string.
func prefixSuccessor(prefix string) string {
	// Attempt to increment the last byte. If that is a 0xff byte, erase it and
	// recurse. If we hit an empty string, then we know our task is impossible.
	limit := []byte(prefix)
	for len(limit) > 0 {
		b := limit[len(limit)-1]
		if b != 0xff {
			limit[len(limit)-1]++
			break
		}

		limit = limit[:len(limit)-1]
	}

	return string(limit)
}

// Return the smallest i such that prefix < s[i].metadata.Name and
// !strings.HasPrefix(s[i].metadata.Name, prefix).
func (s fakeObjectSlice) prefixUpperBound(prefix string) int {
	successor := prefixSuccessor(prefix)
	if successor == "" {
		return len(s)
	}

	return s.lowerBound(successor)
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type bucket struct {
	clock timeutil.Clock
	name  string
	mu    syncutil.InvariantMutex

	// The set of extant objects.
	//
	// INVARIANT: Strictly increasing.
	objects fakeObjectSlice // GUARDED_BY(mu)

	// The most recent generation number that was minted. The next object will
	// receive generation prevGeneration + 1.
	//
	// INVARIANT: This is an upper bound for generation numbers in objects.
	prevGeneration int64 // GUARDED_BY(mu)
}

func checkName(name string) (err error) {
	if len(name) == 0 || len(name) > 1024 {
		err = errors.New("Invalid object name: length must be in [1, 1024]")
		return
	}

	if !utf8.ValidString(name) {
		err = errors.New("Invalid object name: not valid UTF-8")
		return
	}

	for _, r := range name {
		if r == 0x0a || r == 0x0d {
			err = errors.New("Invalid object name: must not contain CR or LF")
			return
		}
	}

	return
}

// LOCKS_REQUIRED(b.mu)
func (b *bucket) checkInvariants() {
	// Make sure 'objects' is strictly increasing.
	for i := 1; i < len(b.objects); i++ {
		objA := b.objects[i-1]
		objB := b.objects[i]
		if !(objA.metadata.Name < objB.metadata.Name) {
			panic(
				fmt.Sprintf(
					"Object names are not strictly increasing: %v vs. %v",
					objA.metadata.Name,
					objB.metadata.Name))
		}
	}

	// Make sure prevGeneration is an upper bound for object generation numbers.
	for _, o := range b.objects {
		if !(o.metadata.Generation <= b.prevGeneration) {
			panic(
				fmt.Sprintf(
					"Object generation %v exceeds %v",
					o.metadata.Generation,
					b.prevGeneration))
		}
	}
}

// Create an object struct for the given attributes and contents.
//
// LOCKS_REQUIRED(b.mu)
func (b *bucket) mintObject(
	req *gcs.CreateObjectRequest,
	contents []byte) (o fakeObject) {
	md5Sum := md5.Sum(contents)
	crc32c := crc32.Checksum(contents, crc32cTable)

	// Set up basic info.
	b.prevGeneration++
	o.metadata = gcs.Object{
		Name:            req.Name,
		ContentType:     req.ContentType,
		ContentLanguage: req.ContentLanguage,
		CacheControl:    req.CacheControl,
		Owner:           "user-fake",
		Size:            uint64(len(contents)),
		ContentEncoding: req.ContentEncoding,
		ComponentCount:  1,
		MD5:             &md5Sum,
		CRC32C:          &crc32c,
		MediaLink:       "http://localhost/download/storage/fake/" + req.Name,
		Metadata:        copyMetadata(req.Metadata),
		Generation:      b.prevGeneration,
		MetaGeneration:  1,
		StorageClass:    "STANDARD",
		Updated:         b.clock.Now(),
	}

	// Set up data.
	o.data = contents

	return
}

// LOCKS_REQUIRED(b.mu)
func (b *bucket) createObjectLocked(
	req *gcs.CreateObjectRequest) (o *gcs.Object, err error) {
	// Check that the name is legal.
	err = checkName(req.Name)
	if err != nil {
		return
	}

	// Snarf the contents.
	contents, err := ioutil.ReadAll(req.Contents)
	if err != nil {
		err = fmt.Errorf("ReadAll: %v", err)
		return
	}

	// Find any existing record for this name.
	existingIndex := b.objects.find(req.Name)

	var existingRecord *fakeObject
	if existingIndex < len(b.objects) {
		existingRecord = &b.objects[existingIndex]
	}

	// Check the provided checksum, if any.
	if req.CRC32C != nil {
		actual := crc32.Checksum(contents, crc32cTable)
		if actual != *req.CRC32C {
			err = fmt.Errorf(
				"CRC32C mismatch: got 0x%08x, expected 0x%08x",
				actual,
				*req.CRC32C)

			return
		}
	}

	// Check the provided hash, if any.
	if req.MD5 != nil {
		actual := md5.Sum(contents)
		if actual != *req.MD5 {
			err = fmt.Errorf(
				"MD5 mismatch: got %s, expected %s",
				hex.EncodeToString(actual[:]),
				hex.EncodeToString(req.MD5[:]))

			return
		}
	}

	// Check preconditions.
	if req.GenerationPrecondition != nil {
		if *req.GenerationPrecondition == 0 && existingRecord != nil {
			err = &gcs.PreconditionError{
				Err: errors.New("Precondition failed: object exists"),
			}

			return
		}

		if *req.GenerationPrecondition > 0 {
			if existingRecord == nil {
				err = &gcs.PreconditionError{
					Err: errors.New("Precondition failed: object doesn't exist"),
				}

				return
			}

			existingGen := existingRecord.metadata.Generation
			if existingGen != *req.GenerationPrecondition {
				err = &gcs.PreconditionError{
					Err: fmt.Errorf(
						"Precondition failed: object has generation %v",
						existingGen),
				}

				return
			}
		}
	}

	if req.MetaGenerationPrecondition != nil {
		if existingRecord == nil {
			err = &gcs.PreconditionError{
				Err: errors.New("Precondition failed: object doesn't exist"),
			}

			return
		}

		existingMetaGen := existingRecord.metadata.MetaGeneration
		if existingMetaGen != *req.MetaGenerationPrecondition {
			err = &gcs.PreconditionError{
				Err: fmt.Errorf(
					"Precondition failed: object has meta-generation %v",
					existingMetaGen),
			}

			return
		}
	}

	// Create an object record from the given attributes.
	var fo fakeObject = b.mintObject(req, contents)
	o = copyObject(&fo.metadata)

	// Replace an entry in or add an entry to our list of objects.
	if existingIndex < len(b.objects) {
		b.objects[existingIndex] = fo
	} else {
		b.objects = append(b.objects, fo)
		sort.Sort(b.objects)
	}

	return
}

// Create a reader based on the supplied request, also returning the index
// within b.objects of the entry for the requested generation.
//
// LOCKS_REQUIRED(b.mu)
func (b *bucket) newReaderLocked(
	req *gcs.ReadObjectRequest) (r io.Reader, index int, err error) {
	// Find the object with the requested name.
	index = b.objects.find(req.Name)
	if index == len(b.objects) {
		err = &gcs.NotFoundError{
			Err: fmt.Errorf("Object %s not found", req.Name),
		}

		return
	}

	o := b.objects[index]

	// Does the generation match?
	if req.Generation != 0 && req.Generation != o.metadata.Generation {
		err = &gcs.NotFoundError{
			Err: fmt.Errorf(
				"Object %s generation %v not found", req.Name, req.Generation),
		}

		return
	}

	// Extract the requested range.
	result := o.data

	if req.Range != nil {
		start := req.Range.Start
		limit := req.Range.Limit
		l := uint64(len(result))

		if start > limit {
			start = 0
			limit = 0
		}

		if start > l {
			start = 0
			limit = 0
		}

		if limit > l {
			limit = l
		}

		result = result[start:limit]
	}

	r = bytes.NewReader(result)

	return
}

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func copyMetadata(in map[string]string) (out map[string]string) {
	if in == nil {
		return
	}

	out = make(map[string]string)
	for k, v := range in {
		out[k] = v
	}

	return
}

func copyObject(o *gcs.Object) *gcs.Object {
	var copy gcs.Object = *o
	copy.Metadata = copyMetadata(o.Metadata)
	return &copy
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

func (b *bucket) Name() string {
	return b.name
}

// LOCKS_EXCLUDED(b.mu)
func (b *bucket) ListObjects(
	ctx context.Context,
	req *gcs.ListObjectsRequest) (listing *gcs.Listing, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Set up the result object.
	listing = new(gcs.Listing)

	// Handle defaults.
	maxResults := req.MaxResults
	if maxResults == 0 {
		maxResults = 1000
	}

	// Find where in the space of object names to start.
	nameStart := req.Prefix
	if req.ContinuationToken != "" && req.ContinuationToken > nameStart {
		nameStart = req.ContinuationToken
	}

	// Find the range of indexes within the array to scan.
	indexStart := b.objects.lowerBound(nameStart)
	prefixLimit := b.objects.prefixUpperBound(req.Prefix)
	indexLimit := minInt(indexStart+maxResults, prefixLimit)

	// Scan the array.
	var lastResultWasPrefix bool
	for i := indexStart; i < indexLimit; i++ {
		var o fakeObject = b.objects[i]
		name := o.metadata.Name

		// Search for a delimiter if necessary.
		if req.Delimiter != "" {
			// Search only in the part after the prefix.
			nameMinusQueryPrefix := name[len(req.Prefix):]

			delimiterIndex := strings.Index(nameMinusQueryPrefix, req.Delimiter)
			if delimiterIndex >= 0 {
				resultPrefixLimit := delimiterIndex

				// Transform to an index within name.
				resultPrefixLimit += len(req.Prefix)

				// Include the delimiter in the result.
				resultPrefixLimit += len(req.Delimiter)

				// Save the result, but only if it's not a duplicate.
				resultPrefix := name[:resultPrefixLimit]
				if len(listing.CollapsedRuns) == 0 ||
					listing.CollapsedRuns[len(listing.CollapsedRuns)-1] != resultPrefix {
					listing.CollapsedRuns = append(listing.CollapsedRuns, resultPrefix)
				}

				isTrailingDelimiter := (delimiterIndex == len(nameMinusQueryPrefix)-1)
				if !isTrailingDelimiter || !req.IncludeTrailingDelimiter {
					lastResultWasPrefix = true
					continue
				}
			}
		}

		lastResultWasPrefix = false

		// Otherwise, return as an object result. Make a copy to avoid handing back
		// internal state.
		listing.Objects = append(listing.Objects, copyObject(&o.metadata))
	}

	// Set up a cursor for where to start the next scan if we didn't exhaust the
	// results.
	if indexLimit < prefixLimit {
		// If the final object we visited was returned as an element in
		// listing.CollapsedRuns, we want to skip all other objects that would
		// result in the same so we don't return duplicate elements in
		// listing.CollapsedRuns across requests.
		if lastResultWasPrefix {
			lastResultPrefix := listing.CollapsedRuns[len(listing.CollapsedRuns)-1]
			listing.ContinuationToken = prefixSuccessor(lastResultPrefix)

			// Check an assumption: prefixSuccessor cannot result in the empty string
			// above because object names must be non-empty UTF-8 strings, and there
			// is no valid non-empty UTF-8 string that consists of entirely 0xff
			// bytes.
			if listing.ContinuationToken == "" {
				err = errors.New("Unexpected empty string from prefixSuccessor")
				return
			}
		} else {
			// Otherwise, we'll start scanning at the next object.
			listing.ContinuationToken = b.objects[indexLimit].metadata.Name
		}
	}

	return
}

// LOCKS_EXCLUDED(b.mu)
func (b *bucket) NewReader(
	ctx context.Context,
	req *gcs.ReadObjectRequest) (rc io.ReadCloser, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	r, _, err := b.newReaderLocked(req)
	if err != nil {
		return
	}

	rc = ioutil.NopCloser(r)
	return
}

// LOCKS_EXCLUDED(b.mu)
func (b *bucket) CreateObject(
	ctx context.Context,
	req *gcs.CreateObjectRequest) (o *gcs.Object, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	o, err = b.createObjectLocked(req)
	return
}

// LOCKS_EXCLUDED(b.mu)
func (b *bucket) CopyObject(
	ctx context.Context,
	req *gcs.CopyObjectRequest) (o *gcs.Object, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check that the destination name is legal.
	err = checkName(req.DstName)
	if err != nil {
		return
	}

	// Does the object exist?
	srcIndex := b.objects.find(req.SrcName)
	if srcIndex == len(b.objects) {
		err = &gcs.NotFoundError{
			Err: fmt.Errorf("Object %q not found", req.SrcName),
		}

		return
	}

	// Does it have the correct generation?
	if req.SrcGeneration != 0 &&
		b.objects[srcIndex].metadata.Generation != req.SrcGeneration {
		err = &gcs.NotFoundError{
			Err: fmt.Errorf(
				"Object %s generation %d not found", req.SrcName, req.SrcGeneration),
		}

		return
	}

	// Does it have the correct meta-generation?
	if req.SrcMetaGenerationPrecondition != nil {
		p := *req.SrcMetaGenerationPrecondition
		if b.objects[srcIndex].metadata.MetaGeneration != p {
			err = &gcs.PreconditionError{
				Err: fmt.Errorf(
					"Object %q has meta-generation %d",
					req.SrcName,
					b.objects[srcIndex].metadata.MetaGeneration),
			}

			return
		}
	}

	// Copy it and assign a new generation number, to ensure that the generation
	// number for the destination name is strictly increasing.
	dst := b.objects[srcIndex]
	dst.metadata.Name = req.DstName
	dst.metadata.MediaLink = "http://localhost/download/storage/fake/" + req.DstName

	b.prevGeneration++
	dst.metadata.Generation = b.prevGeneration

	// Insert into our array.
	existingIndex := b.objects.find(req.DstName)
	if existingIndex < len(b.objects) {
		b.objects[existingIndex] = dst
	} else {
		b.objects = append(b.objects, dst)
		sort.Sort(b.objects)
	}

	o = copyObject(&dst.metadata)
	return
}

// LOCKS_EXCLUDED(b.mu)
func (b *bucket) ComposeObjects(
	ctx context.Context,
	req *gcs.ComposeObjectsRequest) (o *gcs.Object, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// GCS doesn't like too few or too many sources.
	if len(req.Sources) < 1 {
		err = errors.New("You must provide at least one source component")
		return
	}

	if len(req.Sources) > gcs.MaxSourcesPerComposeRequest {
		err = errors.New("You have provided too many source components")
		return
	}

	// Find readers for all of the source objects, also computing the sum of
	// their component counts.
	var srcReaders []io.Reader
	var dstComponentCount int64

	for _, src := range req.Sources {
		var r io.Reader
		var srcIndex int

		r, srcIndex, err = b.newReaderLocked(&gcs.ReadObjectRequest{
			Name:       src.Name,
			Generation: src.Generation,
		})

		if err != nil {
			return
		}

		srcReaders = append(srcReaders, r)
		dstComponentCount += b.objects[srcIndex].metadata.ComponentCount
	}

	// GCS doesn't like the component count to go too high.
	if dstComponentCount > gcs.MaxComponentCount {
		err = errors.New("Result would have too many components")
		return
	}

	// Create the new object.
	createReq := &gcs.CreateObjectRequest{
		Name:                       req.DstName,
		GenerationPrecondition:     req.DstGenerationPrecondition,
		MetaGenerationPrecondition: req.DstMetaGenerationPrecondition,
		Contents:                   io.MultiReader(srcReaders...),
		ContentType:                req.ContentType,
		Metadata:                   req.Metadata,
	}

	_, err = b.createObjectLocked(createReq)
	if err != nil {
		return
	}

	dstIndex := b.objects.find(req.DstName)
	metadata := &b.objects[dstIndex].metadata

	// Touchup: fix the component count.
	metadata.ComponentCount = dstComponentCount

	// Touchup: emulate the real GCS behavior of not exporting an MD5 hash for
	// composite objects.
	metadata.MD5 = nil

	o = copyObject(metadata)
	return
}

// LOCKS_EXCLUDED(b.mu)
func (b *bucket) StatObject(
	ctx context.Context,
	req *gcs.StatObjectRequest) (o *gcs.Object, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Does the object exist?
	index := b.objects.find(req.Name)
	if index == len(b.objects) {
		err = &gcs.NotFoundError{
			Err: fmt.Errorf("Object %s not found", req.Name),
		}

		return
	}

	// Make a copy to avoid handing back internal state.
	o = copyObject(&b.objects[index].metadata)

	return
}

// LOCKS_EXCLUDED(b.mu)
func (b *bucket) UpdateObject(
	ctx context.Context,
	req *gcs.UpdateObjectRequest) (o *gcs.Object, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Does the object exist?
	index := b.objects.find(req.Name)
	if index == len(b.objects) {
		err = &gcs.NotFoundError{
			Err: fmt.Errorf("Object %s not found", req.Name),
		}

		return
	}

	var obj *gcs.Object = &b.objects[index].metadata

	// Does the generation number match the request?
	if req.Generation != 0 && obj.Generation != req.Generation {
		err = &gcs.NotFoundError{
			Err: fmt.Errorf(
				"Object %q generation %d not found",
				req.Name,
				req.Generation),
		}

		return
	}

	// Does the meta-generation precondition check out?
	if req.MetaGenerationPrecondition != nil &&
		obj.MetaGeneration != *req.MetaGenerationPrecondition {
		err = &gcs.PreconditionError{
			Err: fmt.Errorf(
				"Object %q has meta-generation %d",
				obj.Name,
				obj.MetaGeneration),
		}

		return
	}

	// Update the entry's basic fields according to the request.
	if req.ContentType != nil {
		obj.ContentType = *req.ContentType
	}

	if req.ContentEncoding != nil {
		obj.ContentEncoding = *req.ContentEncoding
	}

	if req.ContentLanguage != nil {
		obj.ContentLanguage = *req.ContentLanguage
	}

	if req.CacheControl != nil {
		obj.CacheControl = *req.CacheControl
	}

	// Update the user metadata if necessary.
	if len(req.Metadata) > 0 {
		if obj.Metadata == nil {
			obj.Metadata = make(map[string]string)
		}

		for k, v := range req.Metadata {
			if v == nil {
				delete(obj.Metadata, k)
				continue
			}

			obj.Metadata[k] = *v
		}
	}

	// Bump up the entry generation number and the update time.
	obj.MetaGeneration++
	obj.Updated = b.clock.Now()

	// Make a copy to avoid handing back internal state.
	o = copyObject(obj)

	return
}

// LOCKS_EXCLUDED(b.mu)
func (b *bucket) DeleteObject(
	ctx context.Context,
	req *gcs.DeleteObjectRequest) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Do we possess the object with the given name?
	index := b.objects.find(req.Name)
	if index == len(b.objects) {
		return
	}

	// Don't do anything if the generation is wrong.
	if req.Generation != 0 &&
		b.objects[index].metadata.Generation != req.Generation {
		return
	}

	// Check the meta-generation if requested.
	if req.MetaGenerationPrecondition != nil {
		p := *req.MetaGenerationPrecondition
		if b.objects[index].metadata.MetaGeneration != p {
			err = &gcs.PreconditionError{
				Err: fmt.Errorf(
					"Object %q has meta-generation %d",
					req.Name,
					b.objects[index].metadata.MetaGeneration),
			}

			return
		}
	}

	// Remove the object.
	b.objects = append(b.objects[:index], b.objects[index+1:]...)

	return
}
