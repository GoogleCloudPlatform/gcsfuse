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
	"cloud.google.com/go/storage"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"unicode/utf8"

	"github.com/jacobsa/gcloud/httputil"
	"google.golang.org/api/googleapi"
	storagev1 "google.golang.org/api/storage/v1"

	"golang.org/x/net/context"
)

func (b *bucket) CopyObject(
	ctx context.Context,
	req *CopyObjectRequest) (o *Object, err error) {
	// We encode using json.NewEncoder, which is documented to silently transform
	// invalid UTF-8 (cf. http://goo.gl/3gIUQB). So we can't rely on the server
	// to detect this for us.
	if !utf8.ValidString(req.DstName) {
		err = errors.New("Invalid object name: not valid UTF-8")
		return
	}

	if b.enableStorageClientLibrary {
		o, err = CopyObjectSCL(ctx, req, b.name, b.storageClient)
		return
	}

	// Construct an appropriate URL (cf. https://goo.gl/A41CyJ).
	opaque := fmt.Sprintf(
		"//%s/storage/v1/b/%s/o/%s/copyTo/b/%s/o/%s",
		b.url.Host,
		httputil.EncodePathSegment(b.Name()),
		httputil.EncodePathSegment(req.SrcName),
		httputil.EncodePathSegment(b.Name()),
		httputil.EncodePathSegment(req.DstName))

	query := make(url.Values)
	query.Set("projection", "full")

	if req.SrcGeneration != 0 {
		query.Set("sourceGeneration", fmt.Sprintf("%d", req.SrcGeneration))
	}

	if req.SrcMetaGenerationPrecondition != nil {
		query.Set(
			"ifSourceMetagenerationMatch",
			fmt.Sprintf("%d", *req.SrcMetaGenerationPrecondition))
	}

	if b.billingProject != "" {
		query.Set("userProject", b.billingProject)
	}

	url := &url.URL{
		Scheme:   b.url.Scheme,
		Host:     b.url.Host,
		Opaque:   opaque,
		RawQuery: query.Encode(),
	}

	// Create an HTTP request.
	httpReq, err := httputil.NewRequest(ctx, "POST", url, nil, 0, b.userAgent)
	if err != nil {
		err = fmt.Errorf("httputil.NewRequest: %v", err)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Execute the HTTP request.
	httpRes, err := b.client.Do(httpReq)
	if err != nil {
		return
	}

	defer googleapi.CloseBody(httpRes)

	// Check for HTTP-level errors.
	if err = googleapi.CheckResponse(httpRes); err != nil {
		// Special case: handle not found errors.
		if typed, ok := err.(*googleapi.Error); ok {
			if typed.Code == http.StatusNotFound {
				err = &NotFoundError{Err: typed}
			}
		}

		// Special case: handle precondition errors.
		if typed, ok := err.(*googleapi.Error); ok {
			if typed.Code == http.StatusPreconditionFailed {
				err = &PreconditionError{Err: typed}
			}
		}

		return
	}

	// Parse the response.
	var rawObject *storagev1.Object
	if err = json.NewDecoder(httpRes.Body).Decode(&rawObject); err != nil {
		return
	}

	// Convert the response.
	if o, err = toObject(rawObject); err != nil {
		err = fmt.Errorf("toObject: %v", err)
		return
	}

	return
}

// Custom function to copy Source Object to Destination Object using Storage Client Library.
func CopyObjectSCL(
	ctx context.Context,
	req *CopyObjectRequest, bucketName string, storageClient *storage.Client) (o *Object, err error) {
	// If client is "nil", it means that there was some problem in initializing client in newBucket function of bucket.go file.
	if storageClient == nil {
		err = fmt.Errorf("Error in creating client through Go Storage Library.")
		return
	}

	srcObj := storageClient.Bucket(bucketName).Object(req.SrcName)
	dstObj := storageClient.Bucket(bucketName).Object(req.DstName)

	// Switching to the requested Generation of Source Object.
	if req.SrcGeneration != 0 {
		srcObj = srcObj.Generation(req.SrcGeneration)
	}

	// Putting a condition that the MetaGeneration of source should match *req.SrcMetaGenerationPrecondition for copying operation to occur.
	if req.SrcMetaGenerationPrecondition != nil {
		srcObj = srcObj.If(storage.Conditions{MetagenerationMatch: *req.SrcMetaGenerationPrecondition})
	}

	// Copying Source Object to the Destination Object through a Copier created by Go Storage Client.
	objAttrs, err := dstObj.CopierFrom(srcObj).Run(ctx)
	if err != nil {
		err = fmt.Errorf("Error in copying using Go Storage Client: %v", err)
		return
	}

	// Converting objAttrs to type *Object
	o = ObjectAttrsToBucketObject(objAttrs)

	return
}
