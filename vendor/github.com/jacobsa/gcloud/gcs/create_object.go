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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"unicode/utf8"

	"github.com/jacobsa/gcloud/httputil"
	"golang.org/x/net/context"
	"google.golang.org/api/googleapi"
	storagev1 "google.golang.org/api/storage/v1"
)

// Create the JSON for an "object resource", for use as an Objects.insert body.
func (b *bucket) makeCreateObjectBody(
	req *CreateObjectRequest) (rc io.ReadCloser, err error) {
	// Convert to storagev1.Object.
	rawObject, err := toRawObject(b.Name(), req)
	if err != nil {
		err = fmt.Errorf("toRawObject: %v", err)
		return
	}

	// Serialize.
	j, err := json.Marshal(rawObject)
	if err != nil {
		err = fmt.Errorf("json.Marshal: %v", err)
		return
	}

	// Create a ReadCloser.
	rc = ioutil.NopCloser(bytes.NewReader(j))

	return
}

func (b *bucket) startResumableUpload(
	ctx context.Context,
	req *CreateObjectRequest) (uploadURL *url.URL, err error) {
	// Construct an appropriate URL.
	//
	// The documentation (http://goo.gl/IJSlVK) is extremely vague about how this
	// is supposed to work. As of 2015-03-26, it simply gives an example:
	//
	//     POST https://www.googleapis.com/upload/storage/v1/b/<bucket>/o
	//
	// In Google-internal bug 19718068, it was clarified that the intent is that
	// the bucket name be encoded into a single path segment, as defined by RFC
	// 3986.
	bucketSegment := httputil.EncodePathSegment(b.Name())
	opaque := fmt.Sprintf(
		"//www.googleapis.com/upload/storage/v1/b/%s/o",
		bucketSegment)

	query := make(url.Values)
	query.Set("projection", "full")
	query.Set("uploadType", "resumable")

	if req.GenerationPrecondition != nil {
		query.Set("ifGenerationMatch", fmt.Sprint(*req.GenerationPrecondition))
	}

	if req.MetaGenerationPrecondition != nil {
		query.Set(
			"ifMetagenerationMatch",
			fmt.Sprint(*req.MetaGenerationPrecondition))
	}

	url := &url.URL{
		Scheme:   "https",
		Host:     "www.googleapis.com",
		Opaque:   opaque,
		RawQuery: query.Encode(),
	}

	// Set up the request body.
	body, err := b.makeCreateObjectBody(req)
	if err != nil {
		err = fmt.Errorf("makeCreateObjectBody: %v", err)
		return
	}

	// Create the HTTP request.
	httpReq, err := httputil.NewRequest(
		ctx,
		"POST",
		url,
		body,
		b.userAgent)

	if err != nil {
		err = fmt.Errorf("httputil.NewRequest: %v", err)
		return
	}

	// Set up HTTP request headers.
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Upload-Content-Type", req.ContentType)

	// Execute the HTTP request.
	httpRes, err := b.client.Do(httpReq)
	if err != nil {
		return
	}

	defer googleapi.CloseBody(httpRes)

	// Check for HTTP-level errors.
	if err = googleapi.CheckResponse(httpRes); err != nil {
		return
	}

	// Extract the Location header.
	str := httpRes.Header.Get("Location")
	if str == "" {
		err = fmt.Errorf("Expected a Location header.")
		return
	}

	// Parse it.
	uploadURL, err = url.Parse(str)
	if err != nil {
		err = fmt.Errorf("url.Parse: %v", err)
		return
	}

	return
}

func (b *bucket) CreateObject(
	ctx context.Context,
	req *CreateObjectRequest) (o *Object, err error) {
	// We encode using json.NewEncoder, which is documented to silently transform
	// invalid UTF-8 (cf. http://goo.gl/3gIUQB). So we can't rely on the server
	// to detect this for us.
	if !utf8.ValidString(req.Name) {
		err = errors.New("Invalid object name: not valid UTF-8")
		return
	}

	// Start a resumable upload, obtaining an upload URL.
	uploadURL, err := b.startResumableUpload(ctx, req)
	if err != nil {
		return
	}

	// Set up a follow-up request to the upload URL.
	httpReq, err := httputil.NewRequest(
		ctx,
		"PUT",
		uploadURL,
		ioutil.NopCloser(req.Contents),
		b.userAgent)

	if err != nil {
		err = fmt.Errorf("httputil.NewRequest: %v", err)
		return
	}

	httpReq.Header.Set("Content-Type", req.ContentType)

	// Execute the request.
	httpRes, err := b.client.Do(httpReq)
	if err != nil {
		return
	}

	defer googleapi.CloseBody(httpRes)

	// Check for HTTP-level errors.
	if err = googleapi.CheckResponse(httpRes); err != nil {
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
