// Copyright 2018 Francisco Souza. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package backend

import (
	"fmt"

	"cloud.google.com/go/storage"
)

// ObjectAttrs represents the meta-data without its contents.
type ObjectAttrs struct {
	BucketName      string `json:"-"`
	Name            string `json:"-"`
	Size            int64  `json:"-"`
	ContentType     string
	ContentEncoding string
	Crc32c          string
	Md5Hash         string
	Etag            string
	ACL             []storage.ACLRule
	Metadata        map[string]string
	Created         string
	Deleted         string
	Updated         string
	Generation      int64
}

// ID is used for comparing objects.
func (o *ObjectAttrs) ID() string {
	return fmt.Sprintf("%s#%d", o.IDNoGen(), o.Generation)
}

// IDNoGen does not consider the generation field.
func (o *ObjectAttrs) IDNoGen() string {
	return fmt.Sprintf("%s/%s", o.BucketName, o.Name)
}

// Object represents the object that is stored within the fake server.
type Object struct {
	ObjectAttrs
	Content []byte
}
