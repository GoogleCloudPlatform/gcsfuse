// Copyright 2023 Google LLC
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

package storageutil

import (
	"bytes"
	"io"
	"net/http"
)

// no308Header is set exclusively on resumable upload chunk requests by
// google.golang.org/api/internal/gensupport ((*ResumableUpload).doUploadRequest).
const no308Header = "X-GUploader-No-308"

// WithUserAgent returns a ClientOption that sets the User-Agent. This option is incompatible with the WithHTTPClient option.
// As we are using http-client, we will need to add this header via RoundTripper middleware.
// https://pkg.go.dev/google.golang.org/api/option#WithUserAgent
type userAgentRoundTripper struct {
	wrapped   http.RoundTripper
	UserAgent string
}

func (ug *userAgentRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("User-Agent", ug.UserAgent)
	resp, err := ug.wrapped.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	// Workaround for google.golang.org/api v0.299.0+ regression in
	// internal/gensupport/resumable.go ((*ResumableUpload).transferChunk):
	// When ChunkTransferTimeout is set, transferChunk cancels the per-chunk
	// request context immediately after doUploadRequest returns, before the caller
	// (ObjectsInsertCall.Do -> DecodeResponseBytes) reads the JSON object metadata
	// from resp.Body. For resumable upload chunk requests (identified by the
	// "X-GUploader-No-308: yes" header), eagerly read and close resp.Body while
	// the per-chunk context and its ChunkTransferTimeout timer are still active.
	if r.Header.Get(no308Header) == "yes" && resp != nil && resp.Body != nil {
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		resp.Body = io.NopCloser(bytes.NewReader(body))
	}
	return resp, nil
}
