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

package fake

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"github.com/jacobsa/syncutil"
	"github.com/jacobsa/timeutil"
)

var crc32cTable = crc32.MakeTable(crc32.Castagnoli)

// Equivalent to NewConn(clock).GetBucket(name).
func NewFakeBucket(clock timeutil.Clock, name string, bucketType gcs.BucketType) gcs.Bucket {
	b := &bucket{clock: clock, name: name, bucketType: bucketType}
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

// A slice of folders compared by name.
type fakeFolderSlice []gcs.Folder

func (s fakeObjectSlice) Len() int {
	return len(s)
}

func (f fakeFolderSlice) Len() int {
	return len(f)
}

func (s fakeObjectSlice) Less(i, j int) bool {
	return s[i].metadata.Name < s[j].metadata.Name
}

func (f fakeFolderSlice) Less(i, j int) bool {
	return f[i].Name < f[j].Name
}

func (s fakeObjectSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (f fakeFolderSlice) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

// Return the smallest i such that s[i].metadata.Name >= name, or len(s) if
// there is no such i.
func (s fakeObjectSlice) lowerBound(name string) int {
	pred := func(i int) bool {
		return s[i].metadata.Name >= name
	}

	return sort.Search(len(s), pred)
}

// Return the smallest i such that s[i].Name >= name, or len(s) if
// there is no such i.
func (f fakeFolderSlice) lowerBound(name string) int {
	pred := func(i int) bool {
		return f[i].Name >= name
	}

	return sort.Search(len(f), pred)
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

// Return the smallest i such that f[i].Name == name, or len(s) if
// there is no such i.
func (f fakeFolderSlice) find(name string) int {
	lb := f.lowerBound(name)
	if lb < len(f) && f[lb].Name == name {
		return lb
	}

	return len(f)
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
	clock      timeutil.Clock
	name       string
	bucketType gcs.BucketType
	mu         syncutil.InvariantMutex

	// The set of extant objects.
	//
	// INVARIANT: Strictly increasing.
	objects fakeObjectSlice // GUARDED_BY(mu)
	folders fakeFolderSlice

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

// Create a folder struct for the given name.
//
// LOCKS_REQUIRED(b.mu)
func (b *bucket) mintFolder(folderName string) (f gcs.Folder) {
	f = gcs.Folder{
		Name:       folderName,
		UpdateTime: b.clock.Now(),
	}

	return
}

// In a hierarchical bucket, all directory objects are also retained as folder entries,
// even if we create objects with non-control client API.
// Therefore, whenever we create directory objects in the fake bucket,
// we also need to create a corresponding folder entry for them in HNS.
//
// For example, when creating an object A/B/a.txt where A and B are implicit directories.
// In our existing flow in the fake bucket, we ignore adding entries for A and B.
// In HNS, we have to add these implicit directories as folder entries.
func (b *bucket) addFolderEntry(path string) {
	path = filepath.Dir(path) // Get the directory part of the path
	parts := strings.Split(path, string(filepath.Separator))

	// This is for adding implicit directories as folder entries.
	// For example, createObject(A/B/a.txt) where A and B are implicit directories.
	// We need to add both "A" and "A/B/" as folder entries.
	for i := range parts {
		folder := gcs.Folder{Name: strings.Join(parts[:i+1], string(filepath.Separator)) + string(filepath.Separator)}
		existingIndex := b.folders.find(folder.Name)
		if existingIndex == len(b.folders) {
			b.folders = append(b.folders, folder)
		}
	}
	sort.Sort(b.folders)
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
	contents, err := io.ReadAll(req.Contents)
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

	if b.BucketType() == gcs.Hierarchical {
		b.addFolderEntry(req.Name)
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

func (b *bucket) BucketType() gcs.BucketType {
	return b.bucketType
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

				// In hierarchical buckets, a directory is represented both as a prefix and a folder.
				// Consequently, if a folder entry is discovered, it indicates that it's exclusively a prefix entry.
				//
				// This check was incorporated because createFolder needs to add an entry to the objects, and
				// we cannot distinguish from that entry whether it's solely a prefix.
				//
				// For example, mkdir test will create both a folder entry and a test/ prefix.
				// In our createFolder fake bucket implementation, we created both a folder and an object for
				// the given folderName. There, we can't define whether it's only a prefix and not an object.
				// Hence, we added this check here.
				//
				// Note that in a real ListObject call, the entry will appear only as a prefix and not as an object.
				folderIndex := b.folders.find(resultPrefix)
				if folderIndex < len(b.folders) {
					lastResultWasPrefix = true
					continue
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

	rc = io.NopCloser(r)
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
func (b *bucket) StatObject(ctx context.Context,
	req *gcs.StatObjectRequest) (m *gcs.MinObject, e *gcs.ExtendedObjectAttributes, err error) {
	// If ExtendedObjectAttributes are requested without fetching from gcs enabled, panic.
	if !req.ForceFetchFromGcs && req.ReturnExtendedObjectAttributes {
		panic("invalid StatObjectRequest: ForceFetchFromGcs: false and ReturnExtendedObjectAttributes: true")
	}
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
	o := copyObject(&b.objects[index].metadata)
	m = storageutil.ConvertObjToMinObject(o)
	if req.ReturnExtendedObjectAttributes {
		e = storageutil.ConvertObjToExtendedObjectAttributes(o)
	}
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

func (b *bucket) DeleteFolder(ctx context.Context, folderName string) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Do we possess the folder with the given name?
	index := b.folders.find(folderName)
	if index == len(b.folders) {
		return
	}

	// Remove the folder.
	b.folders = append(b.folders[:index], b.folders[index+1:]...)

	// In the hierarchical bucket, control client API deletes folders as well as
	// prefixes for backward compatibility. Therefore, a prefix object
	// entry needs to be deleted here as well.

	// Do we possess the prefix object with the given name?
	index = b.objects.find(folderName)
	if index == len(b.objects) {
		return
	}

	// Remove the prefix object.
	b.objects = append(b.objects[:index], b.objects[index+1:]...)

	return
}

func (b *bucket) GetFolder(ctx context.Context, foldername string) (*gcs.Folder, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Does the folder exist?
	index := b.folders.find(foldername)
	if index == len(b.folders) {
		err := &gcs.NotFoundError{
			Err: fmt.Errorf("Object %s not found", foldername),
		}
		return nil, err
	}

	return &gcs.Folder{Name: foldername}, nil
}

func (b *bucket) CreateFolder(ctx context.Context, folderName string) (*gcs.Folder, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check that the name is legal.
	err := checkName(folderName)
	if err != nil {
		return nil, err
	}

	// Find any existing record for this name.
	existingIndex := b.folders.find(folderName)

	// Create a folder record from the given attributes.
	fo := b.mintFolder(folderName)

	// Replace an entry in or add an entry to our list of folders.
	if existingIndex < len(b.folders) {
		b.folders[existingIndex] = fo
	} else {
		b.folders = append(b.folders, fo)
		sort.Sort(b.folders)
	}

	// In the hierarchical bucket, control client API creates folders  as well as
	// prefixes for backward compatibility. Therefore, a prefix object
	// entry needs to be created here as well.

	// Find any existing record for this name.
	existingObjectPrefixIndex := b.objects.find(folderName)

	// Create a prefix object record from the given attributes.
	var prefixObject fakeObject
	prefixObject.metadata = gcs.Object{Name: folderName}

	// Replace an entry in or add an entry to our list of objects.
	if existingObjectPrefixIndex < len(b.objects) {
		b.objects[existingObjectPrefixIndex] = prefixObject
	} else {
		b.objects = append(b.objects, prefixObject)
		sort.Sort(b.objects)
	}

	return &fo, nil
}

func (b *bucket) RenameFolder(ctx context.Context, folderName string, destinationFolderId string) (*gcs.Folder, error) {
	// Check that the destination name is legal.
	err := checkName(destinationFolderId)
	if err != nil {
		return nil, err
	}

	// Check if the source folder exists.
	srcIndex := b.folders.find(folderName)
	if srcIndex == len(b.folders) {
		err = &gcs.NotFoundError{
			Err: fmt.Errorf("Object %q not found", folderName),
		}
		return nil, err
	}

	// Find all folders starting with the given prefix and update their names.
	for i := range b.folders {
		if strings.HasPrefix(b.folders[i].Name, folderName) {
			b.folders[i].Name = strings.Replace(b.folders[i].Name, folderName, destinationFolderId, 1)
			b.folders[i].UpdateTime = time.Now()
		}
	}

	// Sort the updated folders.
	sort.Sort(b.folders)

	// Find all objects starting with the given prefix and update their names.
	for i := range b.objects {
		if strings.HasPrefix(b.objects[i].metadata.Name, folderName) {
			b.objects[i].metadata.Name = strings.Replace(b.objects[i].metadata.Name, folderName, destinationFolderId, 1)
			b.objects[i].metadata.Updated = time.Now()
		}
	}

	// Sort the updated objects.
	sort.Sort(b.objects)

	// Return the updated folder.
	folder := &gcs.Folder{
		Name:       destinationFolderId,
		UpdateTime: time.Now(),
	}

	return folder, nil
}
