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

package httputil

// Percent-encode the supplied string so that it matches the grammar laid out
// for the 'segment' category in RFC 3986.
func EncodePathSegment(s string) string {
	// Scan the string once to count how many bytes must be escaped.
	escapeCount := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if shouldEscapeForPathSegment(c) {
			escapeCount++
		}
	}

	// Fast path: is there anything to do?
	if escapeCount == 0 {
		return s
	}

	// Make a buffer that is large enough, then fill it in.
	t := make([]byte, len(s)+2*escapeCount)
	j := 0
	for i := 0; i < len(s); i++ {
		c := s[i]

		// Escape if necessary.
		if shouldEscapeForPathSegment(c) {
			t[j] = '%'
			t[j+1] = "0123456789ABCDEF"[c>>4]
			t[j+2] = "0123456789ABCDEF"[c&15]
			j += 3
		} else {
			t[j] = c
			j++
		}
	}

	return string(t)
}

func shouldEscapeForPathSegment(c byte) bool {
	// According to the following sections of the RFC:
	//
	//     http://tools.ietf.org/html/rfc3986#section-3.3
	//     http://tools.ietf.org/html/rfc3986#section-3.4
	//
	// The grammar for a segment is:
	//
	//     segment       = *pchar
	//     pchar         = unreserved / pct-encoded / sub-delims / ":" / "@"
	//     unreserved    = ALPHA / DIGIT / "-" / "." / "_" / "~"
	//     pct-encoded   = "%" HEXDIG HEXDIG
	//     sub-delims    = "!" / "$" / "&" / "'" / "(" / ")"
	//                   / "*" / "+" / "," / ";" / "="
	//
	// So we need to escape everything that is not in unreserved, sub-delims, or
	// ":" and "@".

	// unreserved (alphanumeric)
	if 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' || '0' <= c && c <= '9' {
		return false
	}

	switch c {
	// unreserved (non-alphanumeric)
	case '-', '.', '_', '~':
		return false

	// sub-delims
	case '!', '$', '&', '\'', '(', ')', '*', '+', ',', ';', '=':
		return false

	// other pchars
	case ':', '@':
		return false
	}

	// Everything else must be escaped.
	return true
}
