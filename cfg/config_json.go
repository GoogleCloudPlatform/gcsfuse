// Copyright 2026 Google LLC
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

package cfg

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"reflect"
)

func sanitizeAndTruncateConfig(c *Config) {
	// Zero out sensitive/local-only paths and tokens to prevent leaking PII in logs
	c.CacheDir = ""
	c.FileSystem.TempDir = ""
	c.FileSystem.KernelParamsFile = ""
	c.GcsAuth.KeyFile = ""
	c.GcsAuth.TokenUrl = ""
	c.GcsConnection.CustomEndpoint = ""
	c.GcsConnection.ExperimentalLocalSocketAddress = ""
	c.Logging.FilePath = ""
	c.Logging.WireLog = ""

	// Truncate long strings recursively
	truncateStructStrings(reflect.ValueOf(c).Elem())
}

func truncateStructStrings(v reflect.Value) {
	if v.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.Kind() == reflect.String && f.CanSet() && f.Len() > 50 {
			f.SetString(f.String()[:49] + "+")
		} else if f.Kind() == reflect.Struct {
			truncateStructStrings(f)
		}
	}
}

// SerializeConfigToGzippedJSONBase64 scrubs the config, serializes it to JSON,
// compresses it using gzip, and encodes it to raw URL-safe base64.
func SerializeConfigToGzippedJSONBase64(c *Config) (string, error) {
	if c == nil {
		return "", nil
	}

	// Copy config by value to avoid mutating the active runtime configuration
	confCopy := *c
	sanitizeAndTruncateConfig(&confCopy)

	jsonBytes, err := json.Marshal(confCopy)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	if _, err := gzipWriter.Write(jsonBytes); err != nil {
		return "", err
	}
	if err := gzipWriter.Close(); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf.Bytes()), nil
}
