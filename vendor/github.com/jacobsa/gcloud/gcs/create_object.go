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
	"cloud.google.com/go/storage"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jacobsa/gcloud/httputil"
	"golang.org/x/net/context"
	"google.golang.org/api/googleapi"
	storagev1 "google.golang.org/api/storage/v1"
)

// Create the JSON for an "object resource", for use as an Objects.insert body.
func (b *bucket) makeCreateObjectBody(
	req *CreateObjectRequest) (body []byte, err error) {
	// Convert to storagev1.Object.
	rawObject, err := toRawObject(b.Name(), req)
	if err != nil {
		err = fmt.Errorf("toRawObject: %v", err)
		return
	}

	// Serialize.
	body, err = json.Marshal(rawObject)
	if err != nil {
		err = fmt.Errorf("json.Marshal: %v", err)
		return
	}

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
		"//%s/upload/storage/v1/b/%s/o",
		b.url.Host,
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

	if b.billingProject != "" {
		query.Set("userProject", b.billingProject)
	}

	url := &url.URL{
		Scheme:   b.url.Scheme,
		Host:     b.url.Host,
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
		ioutil.NopCloser(bytes.NewReader(body)),
		int64(len(body)),
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

	if b.enableStorageClientLibrary {
		o, err = CreateObjectSCL(ctx, req, b.name, b.storageClient)
		return
	}

	// Start a resumable upload, obtaining an upload URL.
	uploadURL, err := b.startResumableUpload(ctx, req)
	if err != nil {
		return
	}

	// Special case: for a few common cases we can explicitly specify a body
	// length, which may assist the HTTP package. In particular, it works around
	// https://golang.org/issue/17071 in versions before Go 1.7.2 when the
	// information is available.
	contentsLength := int64(-1)
	switch v := req.Contents.(type) {
	case *bytes.Buffer:
		contentsLength = int64(v.Len())
	case *bytes.Reader:
		contentsLength = int64(v.Len())
	case *strings.Reader:
		contentsLength = int64(v.Len())
	}

	// Set up a follow-up request to the upload URL.
	httpReq, err := httputil.NewRequest(
		ctx,
		"PUT",
		uploadURL,
		ioutil.NopCloser(req.Contents),
		contentsLength,
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

// Custom function to create an object or upload an existing object to GCS Bucket.
func CreateObjectSCL(
	ctx context.Context,
	req *CreateObjectRequest, bucketName string, storageClient *storage.Client) (o *Object, err error) {
	// If client is "nil", it means that there was some problem in initializing client in newBucket function of bucket.go file.
	if storageClient == nil {
		err = fmt.Errorf("Error in creating client through Go Storage Library.")
		return
	}

	obj := storageClient.Bucket(bucketName).Object(req.Name)

	// Putting conditions on Generation and MetaGeneration of the object for upload to occur.
	if req.GenerationPrecondition != nil {
		if *req.GenerationPrecondition == 0 {
			// Passing because GenerationPrecondition = 0 means object does not exist in the GCS Bucket yet.
		} else if req.MetaGenerationPrecondition != nil && *req.MetaGenerationPrecondition != 0 {
			obj = obj.If(storage.Conditions{GenerationMatch: *req.GenerationPrecondition, MetagenerationMatch: *req.MetaGenerationPrecondition})
		} else {
			obj = obj.If(storage.Conditions{GenerationMatch: *req.GenerationPrecondition})
		}
	}

	// Creating a NewWriter with requested attributes, using Go Storage Client.
	// Chuck size for resumable upload is deafult i.e. 16MB.
	wc := obj.NewWriter(ctx)
	wc.ChunkSize = 0 // This will enable one shot upload and thus increase performance. JSON API Client also performs one-shot upload.
	wc = SetAttrs(wc, req)

	// Copying contents from the request to the Writer. These contents will be copied to the newly created object / already existing object.
	if _, err = io.Copy(wc, req.Contents); err != nil {
		err = fmt.Errorf("Error in io.Copy: %v", err)
		return
	}

	// Closing the Writer.
	if err = wc.Close(); err != nil {
		err = fmt.Errorf("Error in closing writer: %v", err)
		return
	}

	attrs := wc.Attrs() // Retrieving the attributes of the created object.

	// Converting attrs to type *Object.
	o = ObjectAttrsToBucketObject(attrs)

	return
}

// Function for setting attributes to the Writer. These attributes will be assigned to the newly created object / already existing object.
func SetAttrs(wc *storage.Writer, req *CreateObjectRequest) *storage.Writer {
	wc.Name = req.Name
	wc.ContentType = req.ContentType
	wc.ContentLanguage = req.ContentLanguage
	wc.ContentEncoding = req.ContentLanguage
	wc.CacheControl = req.CacheControl
	wc.Metadata = req.Metadata
	wc.ContentDisposition = req.ContentDisposition
	wc.CustomTime, _ = time.Parse(time.RFC3339, req.CustomTime)
	wc.EventBasedHold = req.EventBasedHold
	wc.StorageClass = req.StorageClass

	// Converting []*storagev1.ObjectAccessControl to []ACLRule as expected by the Go Client Writer.
	var Acl []storage.ACLRule
	for _, element := range req.Acl {
		currACL := storage.ACLRule{
			Entity:   storage.ACLEntity(element.Entity),
			EntityID: element.EntityId,
			Role:     storage.ACLRole(element.Role),
			Domain:   element.Domain,
			Email:    element.Email,
			ProjectTeam: &storage.ProjectTeam{
				ProjectNumber: element.ProjectTeam.ProjectNumber,
				Team:          element.ProjectTeam.Team,
			},
		}
		Acl = append(Acl, currACL)
	}
	wc.ACL = Acl

	if req.CRC32C != nil {
		wc.CRC32C = *req.CRC32C
		wc.SendCRC32C = true // Explicitly need to send CRC32C token in Writer in order to send the checksum.
	}

	if req.MD5 != nil {
		wc.MD5 = (*req.MD5)[:]
	}

	return wc
}
