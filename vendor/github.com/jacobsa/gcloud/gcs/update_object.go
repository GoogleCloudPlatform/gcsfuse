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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/jacobsa/gcloud/httputil"
	"golang.org/x/net/context"
	"google.golang.org/api/googleapi"
	storagev1 "google.golang.org/api/storage/v1"
)

func (b *bucket) makeUpdateObjectBody(
	req *UpdateObjectRequest) (rc io.ReadCloser, err error) {
	// Set up a map representing the JSON object we want to send to GCS. For now,
	// we don't treat empty strings specially.
	jsonMap := make(map[string]interface{})

	if req.ContentType != nil {
		jsonMap["contentType"] = req.ContentType
	}

	if req.ContentEncoding != nil {
		jsonMap["contentEncoding"] = req.ContentEncoding
	}

	if req.ContentLanguage != nil {
		jsonMap["contentLanguage"] = req.ContentLanguage
	}

	if req.CacheControl != nil {
		jsonMap["cacheControl"] = req.CacheControl
	}

	// Implement the convention that a pointer to an empty string means to delete
	// the field (communicated to GCS by setting it to null in the JSON).
	for k, v := range jsonMap {
		if *(v.(*string)) == "" {
			jsonMap[k] = nil
		}
	}

	// Add a field for user metadata if appropriate.
	if req.Metadata != nil {
		jsonMap["metadata"] = req.Metadata
	}

	// Set up a reader.
	r, err := googleapi.WithoutDataWrapper.JSONReader(jsonMap)
	if err != nil {
		err = fmt.Errorf("JSONReader", err)
		return
	}

	// Set up a ReadCloser.
	rc = ioutil.NopCloser(r)

	return
}

func (b *bucket) UpdateObject(
	ctx context.Context,
	req *UpdateObjectRequest) (o *Object, err error) {
	// Construct an appropriate URL (cf. http://goo.gl/B46IDy).
	opaque := fmt.Sprintf(
		"//www.googleapis.com/storage/v1/b/%s/o/%s",
		httputil.EncodePathSegment(b.Name()),
		httputil.EncodePathSegment(req.Name))

	query := make(url.Values)
	query.Set("projection", "full")

	if req.Generation != 0 {
		query.Set("generation", fmt.Sprintf("%d", req.Generation))
	}

	if req.MetaGenerationPrecondition != nil {
		query.Set(
			"ifMetagenerationMatch",
			fmt.Sprintf("%d", *req.MetaGenerationPrecondition))
	}

	url := &url.URL{
		Scheme:   "https",
		Host:     "www.googleapis.com",
		Opaque:   opaque,
		RawQuery: query.Encode(),
	}

	// Set up the request body.
	body, err := b.makeUpdateObjectBody(req)
	if err != nil {
		err = fmt.Errorf("makeUpdateObjectBody: %v", err)
		return
	}

	// Create an HTTP request.
	httpReq, err := httputil.NewRequest(ctx, "PATCH", url, body, b.userAgent)
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
