// Copyright 2018 Francisco Souza. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsouza/fake-gcs-server/internal/checksum"
	"github.com/pkg/xattr"
)

// storageFS is an implementation of the backend storage that stores data on disk
//
// The layout is the following:
//
// - rootDir
//
//	|- bucket1
//	\- bucket2
//	  |- object1
//	  \- object2
//
// Bucket and object names are url path escaped, so there's no special meaning of forward slashes.
type storageFS struct {
	rootDir string
	mtx     sync.RWMutex
	mh      metadataHandler
}

// NewStorageFS creates an instance of the filesystem-backed storage backend.
func NewStorageFS(objects []Object, rootDir string) (Storage, error) {
	if !strings.HasSuffix(rootDir, "/") {
		rootDir += "/"
	}
	err := os.MkdirAll(rootDir, 0o700)
	if err != nil {
		return nil, err
	}

	var mh metadataHandler = metadataFile{}
	// Use xattr for metadata if rootDir supports it.
	if xattr.XATTR_SUPPORTED {
		xattrHandler := metadataXattr{}
		var xerr *xattr.Error
		_, err = xattrHandler.read(rootDir)
		if err == nil || (errors.As(err, &xerr) && xerr.Err == xattr.ENOATTR) {
			mh = xattrHandler
		}
	}

	s := &storageFS{rootDir: rootDir, mh: mh}
	for _, o := range objects {
		_, err := s.CreateObject(o)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

// CreateBucket creates a bucket in the fs backend. A bucket is a folder in the
// root directory.
func (s *storageFS) CreateBucket(name string, versioningEnabled bool) error {
	if versioningEnabled {
		return errors.New("not implemented: fs storage type does not support versioning yet")
	}
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return s.createBucket(name)
}

func (s *storageFS) createBucket(name string) error {
	return os.MkdirAll(filepath.Join(s.rootDir, url.PathEscape(name)), 0o700)
}

// ListBuckets returns a list of buckets from the list of directories in the
// root directory.
func (s *storageFS) ListBuckets() ([]Bucket, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	infos, err := os.ReadDir(s.rootDir)
	if err != nil {
		return nil, err
	}
	buckets := []Bucket{}
	for _, info := range infos {
		if info.IsDir() {
			unescaped, err := url.PathUnescape(info.Name())
			if err != nil {
				return nil, fmt.Errorf("failed to unescape object name %s: %w", info.Name(), err)
			}
			buckets = append(buckets, Bucket{Name: unescaped})
		}
	}
	return buckets, nil
}

func timespecToTime(ts syscall.Timespec) time.Time {
	return time.Unix(int64(ts.Sec), int64(ts.Nsec))
}

// GetBucket returns information about the given bucket, or an error if it
// doesn't exist.
func (s *storageFS) GetBucket(name string) (Bucket, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	dirInfo, err := os.Stat(filepath.Join(s.rootDir, url.PathEscape(name)))
	if err != nil {
		return Bucket{}, err
	}
	return Bucket{Name: name, VersioningEnabled: false, TimeCreated: timespecToTime(createTimeFromFileInfo(dirInfo))}, err
}

// DeleteBucket removes the bucket from the backend.
func (s *storageFS) DeleteBucket(name string) error {
	objs, err := s.ListObjects(name, "", false)
	if err != nil {
		return BucketNotFound
	}
	if len(objs) > 0 {
		return BucketNotEmpty
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()
	return os.RemoveAll(filepath.Join(s.rootDir, url.PathEscape(name)))
}

// CreateObject stores an object as a regular file in the disk.
func (s *storageFS) CreateObject(obj Object) (Object, error) {
	if obj.Generation > 0 {
		return Object{}, errors.New("not implemented: fs storage type does not support objects generation yet")
	}

	// Note: this was a quick fix for issue #701. Now that we have a way to
	// persist object attributes, we should implement versioning in the
	// filesystem backend and handle generations outside of the backends.
	obj.Generation = time.Now().UnixNano() / 1000

	s.mtx.Lock()
	defer s.mtx.Unlock()
	err := s.createBucket(obj.BucketName)
	if err != nil {
		return Object{}, err
	}

	path := filepath.Join(s.rootDir, url.PathEscape(obj.BucketName), url.PathEscape(obj.Name))

	if err = os.WriteFile(path, obj.Content, 0o600); err != nil {
		return Object{}, err
	}

	// TODO: Handle if metadata is not present more gracefully?
	encoded, err := json.Marshal(obj.ObjectAttrs)
	if err != nil {
		return Object{}, err
	}

	if err = s.mh.write(path, encoded); err != nil {
		return Object{}, err
	}

	return obj, nil
}

// ListObjects lists the objects in a given bucket with a given prefix and
// delimeter.
func (s *storageFS) ListObjects(bucketName string, prefix string, versions bool) ([]ObjectAttrs, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	infos, err := os.ReadDir(filepath.Join(s.rootDir, url.PathEscape(bucketName)))
	if err != nil {
		return nil, err
	}
	objects := []ObjectAttrs{}
	for _, info := range infos {
		if s.mh.isSpecialFile(info.Name()) {
			continue
		}
		unescaped, err := url.PathUnescape(info.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to unescape object name %s: %w", info.Name(), err)
		}
		if prefix != "" && !strings.HasPrefix(unescaped, prefix) {
			continue
		}
		object, err := s.getObject(bucketName, unescaped)
		if err != nil {
			return nil, err
		}
		object.Size = int64(len(object.Content))
		objects = append(objects, object.ObjectAttrs)
	}
	return objects, nil
}

// GetObject get an object by bucket and name.
func (s *storageFS) GetObject(bucketName, objectName string) (Object, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.getObject(bucketName, objectName)
}

// GetObjectWithGeneration retrieves an specific version of the object. Not
// implemented for this backend.
func (s *storageFS) GetObjectWithGeneration(bucketName, objectName string, generation int64) (Object, error) {
	obj, err := s.GetObject(bucketName, objectName)
	if err != nil {
		return obj, err
	}
	if obj.Generation != generation {
		return obj, fmt.Errorf("generation mismatch, object generation is %v, requested generation is %v (note: filesystem backend does not support versioning)", obj.Generation, generation)
	}
	return obj, nil
}

func (s *storageFS) getObject(bucketName, objectName string) (Object, error) {
	path := filepath.Join(s.rootDir, url.PathEscape(bucketName), url.PathEscape(objectName))

	encoded, err := s.mh.read(path)
	if err != nil {
		return Object{}, err
	}

	var obj Object
	if err = json.Unmarshal(encoded, &obj.ObjectAttrs); err != nil {
		return Object{}, err
	}

	obj.Content, err = os.ReadFile(path)
	if err != nil {
		return Object{}, err
	}

	obj.Name = filepath.ToSlash(objectName)
	obj.BucketName = bucketName
	obj.Size = int64(len(obj.Content))
	return obj, nil
}

// DeleteObject deletes an object by bucket and name.
func (s *storageFS) DeleteObject(bucketName, objectName string) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if objectName == "" {
		return errors.New("can't delete object with empty name")
	}
	path := filepath.Join(s.rootDir, url.PathEscape(bucketName), url.PathEscape(objectName))
	if err := s.mh.remove(path); err != nil {
		return err
	}
	return os.Remove(path)
}

// PatchObject patches the given object metadata.
func (s *storageFS) PatchObject(bucketName, objectName string, metadata map[string]string) (Object, error) {
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

// UpdateObject replaces the given object metadata.
func (s *storageFS) UpdateObject(bucketName, objectName string, metadata map[string]string) (Object, error) {
	obj, err := s.GetObject(bucketName, objectName)
	if err != nil {
		return Object{}, err
	}
	obj.Metadata = map[string]string{}
	for k, v := range metadata {
		obj.Metadata[k] = v
	}
	obj.Generation = 0
	s.CreateObject(obj) // recreate object
	return obj, nil
}

func (s *storageFS) ComposeObject(bucketName string, objectNames []string, destinationName string, metadata map[string]string, contentType string) (Object, error) {
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
		oattrs := ObjectAttrs{
			BucketName:  bucketName,
			Name:        destinationName,
			ContentType: contentType,
			Created:     time.Now().String(),
		}
		dest = Object{
			ObjectAttrs: oattrs,
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
