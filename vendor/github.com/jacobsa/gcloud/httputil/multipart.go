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

// The code in this file is adapted from src/mime/multipart/writer.go in the Go
// repository at commit c007ce824d9a4fccb148f9204e04c23ed2984b71. The LICENSE
// file there says:
//
// Copyright (c) 2012 The Go Authors. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//    * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//    * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//    * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package httputil

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
)

type ContentTypedReader struct {
	ContentType string
	Reader      io.Reader
}

// Create a reader that streams an HTTP multipart body (see RFC 2388) composed
// of the contents of each component reader in sequence, each with a
// Content-Type header as specified.
//
// Unlike multipart.Writer from the standard library, this can be used directly
// as http.Request.Body without bending over backwards to convert an io.Writer
// to an io.Reader.
func NewMultipartReader(ctrs []ContentTypedReader) *MultipartReader {
	boundary := randomBoundary()

	// Read each part followed by the trailer.
	var readers []io.Reader
	for i, ctr := range ctrs {
		readers = append(readers, makePartReader(ctr, i == 0, boundary))
	}

	readers = append(readers, makeTrailerReader(boundary))

	return &MultipartReader{
		boundary: boundary,
		r:        io.MultiReader(readers...),
	}
}

// MultipartReader is an io.Reader that generates HTTP multipart bodies. See
// NewMultipartReader for details.
type MultipartReader struct {
	boundary string
	r        io.Reader
}

func (mr *MultipartReader) Read(p []byte) (n int, err error) {
	n, err = mr.r.Read(p)
	return
}

// Return the value to use for the Content-Type header in an HTTP request that
// uses mr as the body.
func (mr *MultipartReader) ContentType() string {
	return fmt.Sprintf("multipart/related; boundary=%s", mr.boundary)
}

func randomBoundary() string {
	var buf [30]byte
	_, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", buf[:])
}

// Create a reader for a single part and the boundary in front of it. first
// specifies whether this is the first part.
func makePartReader(
	ctr ContentTypedReader,
	first bool,
	boundary string) (r io.Reader) {
	// Set up a buffer containing the boundary.
	var b bytes.Buffer
	if first {
		fmt.Fprintf(&b, "--%s\r\n", boundary)
	} else {
		fmt.Fprintf(&b, "\r\n--%s\r\n", boundary)
	}

	fmt.Fprintf(&b, "Content-Type: %s\r\n", ctr.ContentType)
	fmt.Fprintf(&b, "\r\n")

	// Read the boundary followed by the content.
	r = io.MultiReader(&b, ctr.Reader)

	return
}

// Create a reader for the trailing boundary.
func makeTrailerReader(boundary string) (r io.Reader) {
	var b bytes.Buffer
	fmt.Fprintf(&b, "\r\n--%s--\r\n", boundary)

	r = &b
	return
}
