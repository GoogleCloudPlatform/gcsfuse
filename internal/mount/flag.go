// Copyright 2015 Google LLC
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

package mount

import (
	"strings"
	"time"
)

type ClientProtocol string

const (
	// Deprecated: Use the constant from cfg package
	HTTP1 ClientProtocol = "http1"
	// Deprecated: Use the constant from cfg package
	HTTP2 ClientProtocol = "http2"
	// Deprecated: Use the constant from cfg package
	GRPC ClientProtocol = "grpc"

	// DefaultStatOrTypeCacheTTL is the default value used for
	// stat-cache-ttl or type-cache-ttl if they have not been set
	// by the user.
	DefaultStatOrTypeCacheTTL = time.Minute
	// DefaultStatCacheCapacity is the default value for stat-cache-capacity.
	// This is equivalent of setting metadata-cache: stat-cache-max-size-mb.
	DefaultStatCacheCapacity = 20460

	// DefaultTypeCacheSizeMB is the default value for type-cache-max-size-mb.
	// This is equivalent of setting metadata-cache: type-cache-max-size-mb.
	DefaultTypeCacheSizeMB = 4
)

func (cp ClientProtocol) IsValid() bool {
	switch cp {
	case HTTP1, HTTP2, GRPC:
		return true
	}
	return false
}

// ParseOptions parse an option string in the format accepted by mount(8) and
// generated for its external mount helpers.
//
// It is assumed that option name and values do not contain commas, and that
// the first equals sign in an option is the name/value separator. There is no
// support for escaping.
//
// For example, if the input is
//
//	user,foo=bar=baz,qux
//
// then the following will be inserted into the map.
//
//	"user": "",
//	"foo": "bar=baz",
//	"qux": "",
func ParseOptions(m map[string]string, s string) {
	// NOTE: The man pages don't define how escaping works, and as far
	// as I can tell there is no way to properly escape or quote a comma in the
	// options list for an fstab entry. So put our fingers in our ears and hope
	// that nobody needs a comma.
	for p := range strings.SplitSeq(s, ",") {
		var name string
		var value string

		// Split on the first equals sign.
		if equalsIndex := strings.IndexByte(p, '='); equalsIndex != -1 {
			name = p[:equalsIndex]
			value = p[equalsIndex+1:]
		} else {
			name = p
		}

		m[name] = value
	}

}
