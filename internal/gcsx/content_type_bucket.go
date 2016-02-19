// Copyright 2016 Google Inc. All Rights Reserved.
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

package gcsx

import (
	"mime"
	"path"

	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

// NewContentTypeBucket creates a wrapper bucket that guesses MIME types for
// newly created or composed objects when an explicit type is not already set.
func NewContentTypeBucket(b gcs.Bucket) gcs.Bucket {
	return contentTypeBucket{b}
}

type contentTypeBucket struct {
	gcs.Bucket
}

func (b contentTypeBucket) CreateObject(
	ctx context.Context,
	req *gcs.CreateObjectRequest) (o *gcs.Object, err error) {
	// Guess a content type if necessary.
	if req.ContentType == "" {
		req.ContentType = mime.TypeByExtension(path.Ext(req.Name))
	}

	// Pass on the request.
	o, err = b.Bucket.CreateObject(ctx, req)
	return
}

func (b contentTypeBucket) ComposeObjects(
	ctx context.Context,
	req *gcs.ComposeObjectsRequest) (o *gcs.Object, err error) {
	// Guess a content type if necessary.
	if req.ContentType == "" {
		req.ContentType = mime.TypeByExtension(path.Ext(req.DstName))
	}

	// Pass on the request.
	o, err = b.Bucket.ComposeObjects(ctx, req)
	return
}
