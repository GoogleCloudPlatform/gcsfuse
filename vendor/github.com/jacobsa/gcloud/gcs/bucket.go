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
	"net/http"
	"net/url"

	"github.com/jacobsa/gcloud/httputil"
	"golang.org/x/net/context"
	"google.golang.org/api/googleapi"
	storagev1 "google.golang.org/api/storage/v1"
	"cloud.google.com/go/storage"
    "google.golang.org/api/iterator"
)

// Bucket represents a GCS bucket, pre-bound with a bucket name and necessary
// authorization information.
//
// Each method that may block accepts a context object that is used for
// deadlines and cancellation. Users need not package authorization information
// into the context object (using cloud.WithContext or similar).
//
// All methods are safe for concurrent access.
type Bucket interface {
	Name() string

	// Create a reader for the contents of a particular generation of an object.
	// On a nil error, the caller must arrange for the reader to be closed when
	// it is no longer needed.
	//
	// Non-existent objects cause either this method or the first read from the
	// resulting reader to return an error of type *NotFoundError.
	//
	// Official documentation:
	//     https://cloud.google.com/storage/docs/json_api/v1/objects/get
	NewReader(
		ctx context.Context,
		req *ReadObjectRequest) (io.ReadCloser, error)

	// Create or overwrite an object according to the supplied request. The new
	// object is guaranteed to exist immediately for the purposes of reading (and
	// eventually for listing) after this method returns a nil error. It is
	// guaranteed not to exist before req.Contents returns io.EOF.
	//
	// Official documentation:
	//     https://cloud.google.com/storage/docs/json_api/v1/objects/insert
	//     https://cloud.google.com/storage/docs/json_api/v1/how-tos/upload
	CreateObject(
		ctx context.Context,
		req *CreateObjectRequest) (*Object, error)

	// Copy an object to a new name, preserving all metadata. Any existing
	// generation of the destination name will be overwritten.
	//
	// Returns a record for the new object.
	//
	// Official documentation:
	//     https://cloud.google.com/storage/docs/json_api/v1/objects/copy
	CopyObject(
		ctx context.Context,
		req *CopyObjectRequest) (*Object, error)

	// Compose one or more source objects into a single destination object by
	// concatenating. Any existing generation of the destination name will be
	// overwritten.
	//
	// Returns a record for the new object.
	//
	// Official documentation:
	//     https://cloud.google.com/storage/docs/json_api/v1/objects/compose
	ComposeObjects(
		ctx context.Context,
		req *ComposeObjectsRequest) (*Object, error)

	// Return current information about the object with the given name.
	//
	// Official documentation:
	//     https://cloud.google.com/storage/docs/json_api/v1/objects/get
	StatObject(
		ctx context.Context,
		req *StatObjectRequest) (*Object, error)

    ListMinObjects(
        		ctx context.Context,
        		req *ListObjectsRequest) (*MinObjectListing, error)

	// List the objects in the bucket that meet the criteria defined by the
	// request, returning a result object that contains the results and
	// potentially a cursor for retrieving the next portion of the larger set of
	// results.
	//
	// Official documentation:
	//     https://cloud.google.com/storage/docs/json_api/v1/objects/list
	ListObjects(
		ctx context.Context,
		req *ListObjectsRequest) (*Listing, error)

	// Update the object specified by newAttrs.Name, patching using the non-zero
	// fields of newAttrs.
	//
	// Official documentation:
	//     https://cloud.google.com/storage/docs/json_api/v1/objects/patch
	UpdateObject(
		ctx context.Context,
		req *UpdateObjectRequest) (*Object, error)

	// Delete an object. Non-existence of the object is not treated as an error.
	//
	// Official documentation:
	//     https://cloud.google.com/storage/docs/json_api/v1/objects/delete
	DeleteObject(
		ctx context.Context,
		req *DeleteObjectRequest) error
}

type bucket struct {
	client         *http.Client
	url            *url.URL
	userAgent      string
	name           string
	billingProject string
	objbucket     *storage.BucketHandle
}

func (b *bucket) Name() string {
	return b.name
}


func getProjectionValue(req Projection) storage.Projection {
	// Explicitly converting Projection Value because the ProjectionVal interface of jacobsa/gcloud and Go Client API are not coupled correctly.
	var convertedProjection storage.Projection // Stores the Projection Value according to the Go Client API Interface.
	switch int(req) {
	// Projection Value 0 in jacobsa/gcloud maps to Projection Value 1 in Go Client API, that is for "full".
	case 0:
		convertedProjection = storage.Projection(1)
	// Projection Value 1 in jacobsa/gcloud maps to Projection Value 2 in Go Client API, that is for "noAcl".
	case 1:
		convertedProjection = storage.Projection(2)
	// Default Projection value in jacobsa/gcloud library is 0 that maps to 1 in Go Client API interface, and that is for "full".
	default:
		convertedProjection = storage.Projection(1)
	}
	return convertedProjection
}

func (b *bucket) ListMinObjects(
	ctx context.Context,
	req *ListObjectsRequest) (listing *MinObjectListing, err error) {
	    query := &storage.Query{
                    		Delimiter:                req.Delimiter,
                    		Prefix:                   req.Prefix,
                    		Projection:               getProjectionValue(req.ProjectionVal),
                    		IncludeTrailingDelimiter: req.IncludeTrailingDelimiter,
                    		//MaxResults: , (Field not present in storage.Query of Go Storage Library but present in ListObjectsQuery in Jacobsa code.)
                    	}
        query.SetAttrSelection([]string{"Name","Size","Generation","MetaGeneration","Updated","Metadata"})
        itr := b.objbucket.Objects(ctx, query) // Returning iterator to the list of objects.
                    	pi := itr.PageInfo()
                    	pi.MaxSize = req.MaxResults
                    	pi.Token = req.ContinuationToken
                    	var list MinObjectListing
                    	// Iterating through all the objects in the bucket and one by one adding them to the list.
                                    	for {
                                    		var attrs *storage.ObjectAttrs

                                    		attrs, err = itr.Next()
                                    		if err == iterator.Done {
                                    			err = nil
                                    			break
                                    		}
                                    		if err != nil {
                                    			err = fmt.Errorf("Error in iterating through objects: %w", err)
                                    			return
                                    		}

                                    		// Prefix attribute will be set for the objects returned as part of Prefix[] array in list response.
                                    		// https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/vendor/cloud.google.com/go/storage/storage.go#L1304
                                    		// https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/vendor/cloud.google.com/go/storage/http_client.go#L370
                                    		if attrs.Prefix != "" {
                                    			list.CollapsedRuns = append(list.CollapsedRuns, attrs.Prefix)
                                    		} else {
                                    			// Converting attrs to *Object type.
                                    			currObject := &MinObject{
                                    			    Name : attrs.Name,
                                    			    Size : uint64(attrs.Size),
                                    			    Generation:attrs.Generation,
                                    			    MetaGeneration: attrs.Metageneration,
                                    			    Updated: attrs.Updated,
                                    			    Metadata: attrs.Metadata,
                                    			    }
                                    			list.Objects = append(list.Objects, currObject)
                                    		}

                                    		// itr.next returns all the objects present in the bucket. Hence adding a
                                    		// check to break after iterating over the current page. pi.Remaining()
                                    		// function returns number of items (items + prefixes) remaining in current
                                    		// page to be iterated by iterator (itr). The func returns (number of items in current page - 1)
                                    		// after first itr.Next() call and becomes 0 when iteration is done.
                                    		// If req.MaxResults is 0, then wait till iterator is done. This is similar
                                    		// to https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/vendor/github.com/jacobsa/gcloud/gcs/bucket.go#L164
                                    		if req.MaxResults != 0 && (pi.Remaining() == 0) {
                                    			break
                                    		}
                                    	}

                                    	list.ContinuationToken = itr.PageInfo().Token
                                    	listing = &list
                                    	return
         }
func (b *bucket) ListObjects(
	ctx context.Context,
	req *ListObjectsRequest) (listing *Listing, err error) {
	// Construct an appropriate URL (cf. http://goo.gl/aVSAhT).
	opaque := fmt.Sprintf(
		"//%s/storage/v1/b/%s/o",
		b.url.Host,
		httputil.EncodePathSegment(b.Name()))

	query := make(url.Values)
	query.Set("projection", req.ProjectionVal.String())

	if req.Prefix != "" {
		query.Set("prefix", req.Prefix)
	}

	if req.Delimiter != "" {
		query.Set("delimiter", req.Delimiter)
		query.Set("includeTrailingDelimiter",
			fmt.Sprintf("%v", req.IncludeTrailingDelimiter))
	}

	if req.ContinuationToken != "" {
		query.Set("pageToken", req.ContinuationToken)
	}

	if req.MaxResults != 0 {
		query.Set("maxResults", fmt.Sprintf("%v", req.MaxResults))
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
	httpReq, err := httputil.NewRequest(ctx, "GET", url, nil, 0, b.userAgent)
	if err != nil {
		err = fmt.Errorf("httputil.NewRequest: %v", err)
		return
	}

	// Call the server.
	httpRes, err := b.client.Do(httpReq)
	if err != nil {
		return
	}

	defer googleapi.CloseBody(httpRes)

	// Check for HTTP-level errors.
	if err = googleapi.CheckResponse(httpRes); err != nil {
		return
	}

	// Parse the response.
	var rawListing *storagev1.Objects
	if err = json.NewDecoder(httpRes.Body).Decode(&rawListing); err != nil {
		return
	}

	// Convert the response.
	if listing, err = toListing(rawListing); err != nil {
		return
	}

	return
}

func (b *bucket) StatObject(
	ctx context.Context,
	req *StatObjectRequest) (o *Object, err error) {
	// Construct an appropriate URL (cf. http://goo.gl/MoITmB).
	opaque := fmt.Sprintf(
		"//%s/storage/v1/b/%s/o/%s",
		b.url.Host,
		httputil.EncodePathSegment(b.Name()),
		httputil.EncodePathSegment(req.Name))

	query := make(url.Values)
	query.Set("projection", "full")

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
	httpReq, err := httputil.NewRequest(ctx, "GET", url, nil, 0, b.userAgent)
	if err != nil {
		err = fmt.Errorf("httputil.NewRequest: %v", err)
		return
	}

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

func (b *bucket) DeleteObject(
	ctx context.Context,
	req *DeleteObjectRequest) (err error) {
	// Construct an appropriate URL (cf. http://goo.gl/TRQJjZ).
	opaque := fmt.Sprintf(
		"//%s/storage/v1/b/%s/o/%s",
		b.url.Host,
		httputil.EncodePathSegment(b.Name()),
		httputil.EncodePathSegment(req.Name))

	query := make(url.Values)

	if req.Generation != 0 {
		query.Set("generation", fmt.Sprintf("%d", req.Generation))
	}

	if req.MetaGenerationPrecondition != nil {
		query.Set(
			"ifMetagenerationMatch",
			fmt.Sprintf("%d", *req.MetaGenerationPrecondition))
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
	httpReq, err := httputil.NewRequest(ctx, "DELETE", url, nil, 0, b.userAgent)
	if err != nil {
		err = fmt.Errorf("httputil.NewRequest: %v", err)
		return
	}

	// Execute the HTTP request.
	httpRes, err := b.client.Do(httpReq)
	if err != nil {
		return
	}

	defer googleapi.CloseBody(httpRes)

	// Check for HTTP-level errors.
	err = googleapi.CheckResponse(httpRes)

	// Special case: we want deletes to be idempotent.
	if typed, ok := err.(*googleapi.Error); ok {
		if typed.Code == http.StatusNotFound {
			err = nil
		}
	}

	// Special case: handle precondition errors.
	if typed, ok := err.(*googleapi.Error); ok {
		if typed.Code == http.StatusPreconditionFailed {
			err = &PreconditionError{Err: typed}
		}
	}

	// Propagate other errors.
	if err != nil {
		return
	}

	return
}

func newBucket(
	client *http.Client,
	url *url.URL,
	userAgent string,
	name string,
	billingProject string) Bucket {
	return &bucket{
		client:         client,
		url:            url,
		userAgent:      userAgent,
		name:           name,
		billingProject: billingProject,
	}
}
