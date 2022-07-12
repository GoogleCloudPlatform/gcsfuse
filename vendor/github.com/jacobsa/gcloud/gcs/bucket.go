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
	"crypto/md5"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/jacobsa/gcloud/httputil"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	storagev1 "google.golang.org/api/storage/v1"
	"io"
	"net/http"
	"net/url"
	"time"
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
	storageClient  *storage.Client //Go Storage Library Client.
	url            *url.URL
	userAgent      string
	name           string
	billingProject string
}

func (b *bucket) Name() string {
	return b.name
}

// Convert the object attrs return by the Go Client to Object struct type present in object.go file.
func ObjectAttrsToBucketObject(attrs *storage.ObjectAttrs) *Object {
	// Converting []ACLRule returned by the Go Client into []*storagev1.ObjectAccessControl which complies with GCSFuse type.
	var Acl []*storagev1.ObjectAccessControl
	for _, element := range attrs.ACL {
		currACL := &storagev1.ObjectAccessControl{
			Entity:   string(element.Entity),
			EntityId: element.EntityID,
			Role:     string(element.Role),
			Domain:   element.Domain,
			Email:    element.Email,
			ProjectTeam: &storagev1.ObjectAccessControlProjectTeam{
				ProjectNumber: element.ProjectTeam.ProjectNumber,
				Team:          element.ProjectTeam.Team,
			},
		}
		Acl = append(Acl, currACL)
	}

	// Converting MD5[] slice to MD5[md5.Size] type fixed array as accepted by GCSFuse.
	var MD5 [md5.Size]byte
	copy(MD5[:], attrs.MD5)

	// Setting the parameters in Object and doing conversions as necessary.
	return &Object{
		Name:            attrs.Name,
		ContentType:     attrs.ContentType,
		ContentLanguage: attrs.ContentLanguage,
		CacheControl:    attrs.CacheControl,
		Owner:           attrs.Owner,
		Size:            uint64(attrs.Size),
		ContentEncoding: attrs.ContentEncoding,
		MD5:             &MD5,
		CRC32C:          &attrs.CRC32C,
		MediaLink:       attrs.MediaLink,
		Metadata:        attrs.Metadata,
		Generation:      attrs.Generation,
		MetaGeneration:  attrs.Metageneration,
		StorageClass:    attrs.StorageClass,
		Deleted:         attrs.Deleted,
		Updated:         attrs.Updated,
		//ComponentCount: , (Field not found in attrs returned by Go Client.)
		ContentDisposition: attrs.ContentDisposition,
		CustomTime:         string(attrs.CustomTime.Format(time.RFC3339)),
		EventBasedHold:     attrs.EventBasedHold,
		Acl:                Acl,
	}
}

func (b *bucket) ListObjects(
	ctx context.Context,
	req *ListObjectsRequest) (listing *Listing, err error) {
	if true {
		listing, err = ListObjectsSCL(ctx, req, b.name, b.storageClient)
		return
	}

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
	fmt.Println((*listing))
	return
}

// Custom function to list objects in a bucket using Storage Client Library.
func ListObjectsSCL(
	ctx context.Context,
	req *ListObjectsRequest, bucketName string, storageClient *storage.Client) (listing *Listing, err error) {
	// If client is "nil", it means that there was some problem in initializing client in newBucket function of bucket.go file.
	if storageClient == nil {
		err = fmt.Errorf("Error in creating client through Go Storage Library.")
		return
	}

	// Converting *ListObjectsRequest to type *storage.Query as expected by the Go Storage Client.
	query := &storage.Query{
		Delimiter:                req.Delimiter,
		Prefix:                   req.Prefix,
		Projection:               storage.Projection(req.ProjectionVal),
		IncludeTrailingDelimiter: req.IncludeTrailingDelimiter,
		//MaxResults: , (Field not present in storage.Query of Go Storage Library but present in ListObjectsQuery in Jacobsa code.)
	}
	itr := storageClient.Bucket(bucketName).Objects(ctx, query) // Returning iterator to the list of objects.
	var list Listing

	// Iterating through all the objects in the bucket and one by one adding them to the list.
	for {
		var attrs *storage.ObjectAttrs = nil
		attrs, err = itr.Next()
		if err == iterator.Done {
			err = nil
			break
		}
		if err != nil {
			err = fmt.Errorf("Error in iterating through objects: %v", err)
			return
		}

		// Converting attrs to *Object type.
		currObject := ObjectAttrsToBucketObject(attrs)
		list.Objects = append(list.Objects, currObject)
	}

	listing = &list
	return
}

func (b *bucket) StatObject(
	ctx context.Context,
	req *StatObjectRequest) (o *Object, err error) {
	if true {
		o, err = StatObjectSCL(ctx, req, b.name, b.storageClient)
		return
	}
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

// Custom function made to return the attributes of an object using Storage Client Library.
func StatObjectSCL(
	ctx context.Context,
	req *StatObjectRequest, bucketName string, storageClient *storage.Client) (o *Object, err error) {
	// If client is "nil", it means that there was some problem in initializing client in newBucket function of bucket.go file.
	if storageClient == nil {
		err = fmt.Errorf("Error in creating client through Go Storage Library.")
		return
	}

	var attrs *storage.ObjectAttrs = nil
	// Retrieving object attrs through Go Storage Client.
	attrs, err = storageClient.Bucket(bucketName).Object(req.Name).Attrs(ctx)

	// If error is of type storage.ErrObjectNotExist, then we have to retry once by appending '/' to the object name.
	// We are retyring to handle the case when the object is a directory.
	// Since directories in GCS bucket are denoted with a an extra '/' at the end of their name. But in the request we are only provided with their name without '/'.
	if err == storage.ErrObjectNotExist {
		dirName := req.Name + "/"
		attrs, err = storageClient.Bucket(bucketName).Object(dirName).Attrs(ctx)
		if err == storage.ErrObjectNotExist {
			err = &NotFoundError{Err: err} // Special case error that object not found in the bucket.
			return
		}
	}
	if err != nil {
		err = fmt.Errorf("Error in returning object attributes: %v", err)
		return
	}

	// Converting attrs to type *Object
	o = ObjectAttrsToBucketObject(attrs)

	return
}

func (b *bucket) DeleteObject(
	ctx context.Context,
	req *DeleteObjectRequest) (err error) {
	if true {
		err = DeleteObjectSCL(ctx, req, b.name, b.storageClient)
		return
	}

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

// Custom function made to delete an object using Storage Client Library.
func DeleteObjectSCL(
	ctx context.Context,
	req *DeleteObjectRequest, bucketName string, storageClient *storage.Client) (err error) {
	// If client is "nil", it means that there was some problem in initializing client in newBucket function of bucket.go file.
	if storageClient == nil {
		err = fmt.Errorf("Error in creating client through Go Storage Library.")
		return
	}

	obj := storageClient.Bucket(bucketName).Object(req.Name)

	// Switching to the requested generation of the object.
	if req.Generation != 0 {
		obj = obj.Generation(req.Generation)
	}

	// Putting condition that the object's MetaGeneration should match the requested MetaGeneration for deletion to occur.
	if req.MetaGenerationPrecondition != nil && *req.MetaGenerationPrecondition != 0 {

		obj = obj.If(storage.Conditions{MetagenerationMatch: *req.MetaGenerationPrecondition})

	}

	// Deleting object through Go Storage Client.
	err = obj.Delete(ctx)
	if err != nil {
		err = fmt.Errorf("Error in deleting the object through Go storage client: %v", err)
		return
	}

	return
}

func newBucket(
	ctx context.Context,
	client *http.Client,
	url *url.URL,
	userAgent string,
	name string,
	billingProject string,
	tokenSrc oauth2.TokenSource,
	goClientConfig *GoClientConfig) (b Bucket, err error) {

	// Creating client through Go Storage Client Library for the storageClient parameter of bucket.
	var tr *http.Transport = nil

	// Choosing between HTTP1 and HTTP2.
	if goClientConfig.DisableHTTP2 {
		tr = &http.Transport{
			MaxConnsPerHost:     goClientConfig.MaxConnsPerHost,
			MaxIdleConnsPerHost: goClientConfig.MaxIdleConnsPerHost,
			// This disables HTTP/2 in transport.
			TLSNextProto: make(
				map[string]func(string, *tls.Conn) http.RoundTripper,
			),
		}
	} else {
		tr = &http.Transport{
			DisableKeepAlives: true,
			MaxConnsPerHost:   goClientConfig.MaxConnsPerHost, // Not affecting the performance when HTTP 2.0 is enabled.
			ForceAttemptHTTP2: true,
		}
	}

	// Custom http Client for Go Client.
	httpClient := &http.Client{Transport: &oauth2.Transport{
		Base:   tr,
		Source: tokenSrc,
	},
		Timeout: 2 * time.Second,
	}

	var storageClient *storage.Client = nil
	storageClient, err = storage.NewClient(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		err = fmt.Errorf("Error in creating the client through Go Storage Library: %v", err)
	}

	b = &bucket{
		client:         client,
		storageClient:  storageClient,
		url:            url,
		userAgent:      userAgent,
		name:           name,
		billingProject: billingProject,
	}
	return
}
