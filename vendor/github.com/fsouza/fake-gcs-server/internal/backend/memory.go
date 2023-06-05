// Copyright 2018 Francisco Souza. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package backend

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fsouza/fake-gcs-server/internal/checksum"
)

const timestampFormat = "2006-01-02T15:04:05.999999Z07:00"

// storageMemory is an implementation of the backend storage that stores data
// in memory.
type storageMemory struct {
	buckets map[string]bucketInMemory
	mtx     sync.RWMutex
}

type bucketInMemory struct {
	Bucket
	// maybe we can refactor how the memory backend works? no need to store
	// Object instances.
	activeObjects   []Object
	archivedObjects []Object
}

func newBucketInMemory(name string, versioningEnabled bool) bucketInMemory {
	return bucketInMemory{Bucket{name, versioningEnabled, time.Now()}, []Object{}, []Object{}}
}

func (bm *bucketInMemory) addObject(obj Object) Object {
	obj.Size = int64(len(obj.Content))
	obj.Generation = getNewGenerationIfZero(obj.Generation)
	index := findObject(obj, bm.activeObjects, false)
	if index >= 0 {
		if bm.VersioningEnabled {
			bm.activeObjects[index].Deleted = time.Now().Format(timestampFormat)
			bm.cpToArchive(bm.activeObjects[index])
		}
		bm.activeObjects[index] = obj
	} else {
		bm.activeObjects = append(bm.activeObjects, obj)
	}

	return obj
}

func getNewGenerationIfZero(generation int64) int64 {
	if generation == 0 {
		return time.Now().UnixNano() / 1000
	}
	return generation
}

func (bm *bucketInMemory) deleteObject(obj Object, matchGeneration bool) {
	index := findObject(obj, bm.activeObjects, matchGeneration)
	if index < 0 {
		return
	}
	if bm.VersioningEnabled {
		obj.Deleted = time.Now().Format(timestampFormat)
		bm.mvToArchive(obj)
	} else {
		bm.deleteFromObjectList(obj, true)
	}
}

func (bm *bucketInMemory) cpToArchive(obj Object) {
	bm.archivedObjects = append(bm.archivedObjects, obj)
}

func (bm *bucketInMemory) mvToArchive(obj Object) {
	bm.cpToArchive(obj)
	bm.deleteFromObjectList(obj, true)
}

func (bm *bucketInMemory) deleteFromObjectList(obj Object, active bool) {
	objects := bm.activeObjects
	if !active {
		objects = bm.archivedObjects
	}
	index := findObject(obj, objects, !active)
	objects[index] = objects[len(objects)-1]
	if active {
		bm.activeObjects = objects[:len(objects)-1]
	} else {
		bm.archivedObjects = objects[:len(objects)-1]
	}
}

// findObject looks for an object in the given list and return the index where it
// was found, or -1 if the object doesn't exist.
func findObject(obj Object, objectList []Object, matchGeneration bool) int {
	for i, o := range objectList {
		if matchGeneration && obj.ID() == o.ID() {
			return i
		}
		if !matchGeneration && obj.IDNoGen() == o.IDNoGen() {
			return i
		}
	}
	return -1
}

// NewStorageMemory creates an instance of StorageMemory.
func NewStorageMemory(objects []Object) Storage {
	s := &storageMemory{
		buckets: make(map[string]bucketInMemory),
	}
	for _, o := range objects {
		s.CreateBucket(o.BucketName, false)
		bucket := s.buckets[o.BucketName]
		bucket.addObject(o)
		s.buckets[o.BucketName] = bucket
	}
	return s
}

// CreateBucket creates a bucket.
func (s *storageMemory) CreateBucket(name string, versioningEnabled bool) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	bucket, err := s.getBucketInMemory(name)
	if err == nil {
		if bucket.VersioningEnabled != versioningEnabled {
			return fmt.Errorf("a bucket named %s already exists, but with different properties", name)
		}
		return nil
	}
	s.buckets[name] = newBucketInMemory(name, versioningEnabled)
	return nil
}

// ListBuckets lists buckets currently registered in the backend.
func (s *storageMemory) ListBuckets() ([]Bucket, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	buckets := []Bucket{}
	for _, bucketInMemory := range s.buckets {
		buckets = append(buckets, Bucket{bucketInMemory.Name, bucketInMemory.VersioningEnabled, bucketInMemory.TimeCreated})
	}
	return buckets, nil
}

// GetBucket retrieves the bucket information from the backend.
func (s *storageMemory) GetBucket(name string) (Bucket, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	bucketInMemory, err := s.getBucketInMemory(name)
	return Bucket{bucketInMemory.Name, bucketInMemory.VersioningEnabled, bucketInMemory.TimeCreated}, err
}

func (s *storageMemory) getBucketInMemory(name string) (bucketInMemory, error) {
	if bucketInMemory, found := s.buckets[name]; found {
		return bucketInMemory, nil
	}
	return bucketInMemory{}, fmt.Errorf("no bucket named %s", name)
}

// DeleteBucket removes the bucket from the backend.
func (s *storageMemory) DeleteBucket(name string) error {
	objs, err := s.ListObjects(name, "", false)
	if err != nil {
		return BucketNotFound
	}
	if len(objs) > 0 {
		return BucketNotEmpty
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()
	delete(s.buckets, name)
	return nil
}

// CreateObject stores an object in the backend.
func (s *storageMemory) CreateObject(obj Object) (Object, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	bucketInMemory, err := s.getBucketInMemory(obj.BucketName)
	if err != nil {
		bucketInMemory = newBucketInMemory(obj.BucketName, false)
	}
	newObj := bucketInMemory.addObject(obj)
	s.buckets[obj.BucketName] = bucketInMemory
	return newObj, nil
}

// ListObjects lists the objects in a given bucket with a given prefix and
// delimeter.
func (s *storageMemory) ListObjects(bucketName string, prefix string, versions bool) ([]ObjectAttrs, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	bucketInMemory, err := s.getBucketInMemory(bucketName)
	if err != nil {
		return []ObjectAttrs{}, err
	}
	objAttrs := make([]ObjectAttrs, 0, len(bucketInMemory.activeObjects))
	for _, obj := range bucketInMemory.activeObjects {
		if prefix != "" && !strings.HasPrefix(obj.Name, prefix) {
			continue
		}
		objAttrs = append(objAttrs, obj.ObjectAttrs)
	}
	if !versions {
		return objAttrs, nil
	}

	archvObjs := make([]ObjectAttrs, 0, len(bucketInMemory.archivedObjects))
	for _, obj := range bucketInMemory.archivedObjects {
		if prefix != "" && !strings.HasPrefix(obj.Name, prefix) {
			continue
		}
		archvObjs = append(archvObjs, obj.ObjectAttrs)
	}
	return append(objAttrs, archvObjs...), nil
}

func (s *storageMemory) GetObject(bucketName, objectName string) (Object, error) {
	return s.GetObjectWithGeneration(bucketName, objectName, 0)
}

// GetObjectWithGeneration retrieves a specific version of the object.
func (s *storageMemory) GetObjectWithGeneration(bucketName, objectName string, generation int64) (Object, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	bucketInMemory, err := s.getBucketInMemory(bucketName)
	if err != nil {
		return Object{}, err
	}
	matchGeneration := false
	obj := Object{ObjectAttrs: ObjectAttrs{BucketName: bucketName, Name: objectName}}
	listToConsider := bucketInMemory.activeObjects
	if generation != 0 {
		matchGeneration = true
		obj.Generation = generation
		listToConsider = append(listToConsider, bucketInMemory.archivedObjects...)
	}
	index := findObject(obj, listToConsider, matchGeneration)
	if index < 0 {
		return obj, errors.New("object not found")
	}

	return listToConsider[index], nil
}

func (s *storageMemory) DeleteObject(bucketName, objectName string) error {
	obj, err := s.GetObject(bucketName, objectName)
	if err != nil {
		return err
	}
	s.mtx.Lock()
	defer s.mtx.Unlock()
	bucketInMemory, err := s.getBucketInMemory(bucketName)
	if err != nil {
		return err
	}
	bucketInMemory.deleteObject(obj, true)
	s.buckets[bucketName] = bucketInMemory
	return nil
}

// PatchObject updates an object metadata.
func (s *storageMemory) PatchObject(bucketName, objectName string, metadata map[string]string) (Object, error) {
	obj, err := s.GetObject(bucketName, objectName)
	if err != nil {
		return Object{}, err
	}
	if obj.Metadata == nil {
		obj.Metadata = map[string]string{}
	}
	for k, v := range metadata {
		obj.Metadata[k] = v
	}
	s.CreateObject(obj) // recreate object
	return obj, nil
}

// UpdateObject replaces an object metadata.
func (s *storageMemory) UpdateObject(bucketName, objectName string, metadata map[string]string) (Object, error) {
	obj, err := s.GetObject(bucketName, objectName)
	if err != nil {
		return Object{}, err
	}
	obj.Metadata = map[string]string{}
	for k, v := range metadata {
		obj.Metadata[k] = v
	}
	s.CreateObject(obj) // recreate object
	return obj, nil
}

func (s *storageMemory) ComposeObject(bucketName string, objectNames []string, destinationName string, metadata map[string]string, contentType string) (Object, error) {
	var data []byte
	for _, n := range objectNames {
		obj, err := s.GetObject(bucketName, n)
		if err != nil {
			return Object{}, err
		}
		data = append(data, obj.Content...)
	}

	dest, err := s.GetObject(bucketName, destinationName)
	if err != nil {
		dest = Object{
			ObjectAttrs: ObjectAttrs{
				BucketName:  bucketName,
				Name:        destinationName,
				ContentType: contentType,
				Created:     time.Now().String(),
			},
		}
	}

	dest.Content = data
	dest.Crc32c = checksum.EncodedCrc32cChecksum(data)
	dest.Md5Hash = checksum.EncodedMd5Hash(data)
	dest.Metadata = metadata

	result, err := s.CreateObject(dest)
	if err != nil {
		return result, err
	}

	return result, nil
}
