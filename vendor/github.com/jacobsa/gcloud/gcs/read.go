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
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"strings"

	"github.com/jacobsa/gcloud/httputil"
	"golang.org/x/net/context"
	"google.golang.org/api/googleapi"
)

func (b *bucket) NewReader(
	ctx context.Context,
	req *ReadObjectRequest) (rc io.ReadCloser, err error) {
	// Construct an appropriate URL.
	//
	// The documentation (https://goo.gl/9zeA98) is vague about how this is
	// supposed to work. As of 2015-05-14, it has no prose but gives the example:
	//
	//     www.googleapis.com/download/storage/v1/b/<bucket>/o/<object>?alt=media
	//
	// In Google-internal bug 19718068, it was clarified that the intent is that
	// each of the bucket and object names are encoded into a single path
	// segment, as defined by RFC 3986.
	bucketSegment := httputil.EncodePathSegment(b.name)
	objectSegment := httputil.EncodePathSegment(req.Name)
	opaque := fmt.Sprintf(
		"//%s/download/storage/v1/b/%s/o/%s",
		b.url.Host,
		bucketSegment,
		objectSegment)

	query := make(url.Values)
	query.Set("alt", "media")

	if req.Generation != 0 {
		query.Set("generation", fmt.Sprintf("%d", req.Generation))
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

	// Set a Range header, if appropriate.
	var bodyLimit int64
	if req.Range != nil {
		var v string
		v, bodyLimit = makeRangeHeaderValue(*req.Range)
		httpReq.Header.Set("Range", v)
	}

	// Call the server.
	httpRes, err := b.client.Do(httpReq)
	if err != nil {
		return
	}

	// Close the body if we're returning in error.
	defer func() {
		if err != nil {
			googleapi.CloseBody(httpRes)
		}
	}()

	// Check for HTTP error statuses.
	if err = googleapi.CheckResponse(httpRes); err != nil {
		if typed, ok := err.(*googleapi.Error); ok {
			// Special case: handle not found errors.
			if typed.Code == http.StatusNotFound {
				err = &NotFoundError{Err: typed}
			}

			// Special case: if the user requested a range and we received HTTP 416
			// from the server, treat this as an empty body. See makeRangeHeaderValue
			// for more details.
			if req.Range != nil &&
				typed.Code == http.StatusRequestedRangeNotSatisfiable {
				err = nil
				googleapi.CloseBody(httpRes)
				rc = ioutil.NopCloser(strings.NewReader(""))
			}
		}

		return
	}

	// The body contains the object data.
	rc = httpRes.Body

	// If the user requested a range and we didn't see HTTP 416 above, we require
	// an HTTP 206 response and must truncate the body. See the notes on
	// makeRangeHeaderValue.
	if req.Range != nil {
		if httpRes.StatusCode != http.StatusPartialContent {
			err = fmt.Errorf(
				"Received unexpected status code %d instead of HTTP 206",
				httpRes.StatusCode)

			return
		}

		rc = newLimitReadCloser(rc, bodyLimit)
	}

	return
}

// Given a [start, limit) range, create an HTTP 1.1 Range header which ensures
// that the resulting body is what the user intended, given the following
// protocol:
//
//  *  If GCS returns HTTP 416 (Requested range not satisfiable), treat the
//     body as if it is empty.
//
//  *  If GCS returns HTTP 206 (Partial Content), truncate the body to at most
//     the returned number of bytes. Do not treat the body already being
//     shorter than this length as an error.
//
//  *  If GCS returns any other status code, regard it as an error.
//
// This monkeying around is necessary because of various shitty aspects of the
// HTTP 1.1 Range header:
//
//  *  Ranges are [min, max] and require min <= max.
//
//  *  Therefore users cannot request empty ranges, even though the status code
//     and headers may still be meaningful without actual entity body content.
//
//  *  min must be less than the entity body length, unless the server is
//     polite enough to send HTTP 416 when this is not the case. Luckily GCS
//     appears to be.
//
// Cf. http://tools.ietf.org/html/rfc2616#section-14.35.1
func makeRangeHeaderValue(br ByteRange) (hdr string, n int64) {
	// HACK(jacobsa): Above a certain number N, GCS appears to treat Range
	// headers containing a last-byte-pos > N as syntactically invalid. I've
	// experimentally determined that N is 2^63-1, which makes sense if they are
	// using signed integers.
	//
	// Since math.MaxUint64 is a reasonable way to express "infinity" for a
	// limit, and because we don't intend to support eight-exabyte objects,
	// handle this by truncating the limit. This also prevents overflow when
	// casting to int64 below.
	if br.Limit > math.MaxInt64 {
		br.Limit = math.MaxInt64
	}

	// Canonicalize ranges that the server will not like. We must do this because
	// RFC 2616 §14.35.1 requires the last byte position to be greater than or
	// equal to the first byte position.
	if br.Limit < br.Start {
		br.Start = 0
		br.Limit = 0
	}

	// HTTP byte range specifiers are [min, max] double-inclusive, ugh. But we
	// require the user to truncate, so there is no harm in requesting one byte
	// extra at the end. If the range GCS sees goes past the end of the object,
	// it truncates. If the range starts after the end of the object, it returns
	// HTTP 416, which we require the user to handle.
	hdr = fmt.Sprintf("bytes=%d-%d", br.Start, br.Limit)
	n = int64(br.Limit - br.Start)

	return
}

type limitReadCloser struct {
	n       int64
	wrapped io.ReadCloser
}

func (lrc *limitReadCloser) Read(p []byte) (n int, err error) {
	// Clip the read to what's left to return.
	if int64(len(p)) > lrc.n {
		p = p[:lrc.n]
	}

	// If there's a read left to do, perform it.
	//
	// If the wrapped reader returns an error, we want to pass it through
	// directly without monkeying around with draining it further. This includes
	// the usual case of EOF, since we don't need to drain it in that case.
	n, err = lrc.wrapped.Read(p)
	lrc.n -= int64(n)

	if err != nil {
		return
	}

	// If we're not yet done with the range of interest, there's nothing further
	// to do.
	if lrc.n > 0 {
		return
	}

	// Drain the rest of the contents from the wrapped reader and return EOF.
	_, err = io.Copy(ioutil.Discard, lrc.wrapped)
	if err != nil {
		err = fmt.Errorf("Discarding additional contents: %v", err)
		return
	}

	err = io.EOF
	return
}

func (lrc *limitReadCloser) Close() (err error) {
	err = lrc.wrapped.Close()
	return
}

// Create an io.ReadCloser that limits the amount of data returned by a wrapped
// io.ReadCloser. Additional data is read up to EOF but discarded, making it
// suitable for use with http.Response.Body which must be drained fully in
// order to reuse the keep-alive connection.
func newLimitReadCloser(wrapped io.ReadCloser, n int64) (rc io.ReadCloser) {
	rc = &limitReadCloser{
		n:       n,
		wrapped: wrapped,
	}

	return
}
