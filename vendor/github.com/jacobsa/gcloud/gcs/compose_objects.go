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
	"google.golang.org/api/googleapi"
	storagev1 "google.golang.org/api/storage/v1"

	"golang.org/x/net/context"
)

func (b *bucket) makeComposeObjectsBody(
	req *ComposeObjectsRequest) (rc io.ReadCloser, err error) {
	// Create a request in the form expected by the API.
	r := storagev1.ComposeRequest{
		Destination: &storagev1.Object{
			Name:     req.DstName,
			Metadata: req.Metadata,

			// We get an HTTP 400 if we don't set this.
			// Cf. Google-internal bug 21588058.
			ContentType: "application/octet-stream",
		},
	}

	for _, src := range req.Sources {
		s := &storagev1.ComposeRequestSourceObjects{
			Name:       src.Name,
			Generation: src.Generation,
		}

		r.SourceObjects = append(r.SourceObjects, s)
	}

	// Serialize it.
	j, err := json.Marshal(&r)
	if err != nil {
		err = fmt.Errorf("json.Marshal: %v", err)
		return
	}

	// Create a ReadCloser.
	rc = ioutil.NopCloser(bytes.NewReader(j))

	return
}

func (b *bucket) ComposeObjects(
	ctx context.Context,
	req *ComposeObjectsRequest) (o *Object, err error) {
	// We encode using json.NewEncoder, which is documented to silently transform
	// invalid UTF-8 (cf. http://goo.gl/3gIUQB). So we can't rely on the server
	// to detect this for us.
	if !utf8.ValidString(req.DstName) {
		err = errors.New("Invalid object name: not valid UTF-8")
		return
	}

	// Construct an appropriate URL.
	bucketSegment := httputil.EncodePathSegment(b.Name())
	objectSegment := httputil.EncodePathSegment(req.DstName)

	opaque := fmt.Sprintf(
		"//www.googleapis.com/storage/v1/b/%s/o/%s/compose",
		bucketSegment,
		objectSegment)

	query := make(url.Values)
	if req.DstGenerationPrecondition != nil {
		query.Set("ifGenerationMatch", fmt.Sprint(*req.DstGenerationPrecondition))
	}

	url := &url.URL{
		Scheme:   "https",
		Host:     "www.googleapis.com",
		Opaque:   opaque,
		RawQuery: query.Encode(),
	}

	// Set up the request body.
	body, err := b.makeComposeObjectsBody(req)
	if err != nil {
		err = fmt.Errorf("makeComposeObjectsBody: %v", err)
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

	// Execute the HTTP request.
	httpRes, err := b.client.Do(httpReq)
	if err != nil {
		return
	}

	defer googleapi.CloseBody(httpRes)

	// Check for HTTP-level errors.
	if err = googleapi.CheckResponse(httpRes); err != nil {
		// Special case: handle not found and precondition errors.
		if typed, ok := err.(*googleapi.Error); ok {
			switch typed.Code {
			case http.StatusNotFound:
				err = &NotFoundError{Err: typed}

			case http.StatusPreconditionFailed:
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
