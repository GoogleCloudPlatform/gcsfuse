// Copyright 2024 Google LLC
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

import "net/url"

func decodeURL(u string) (string, error) {
	// TODO: check if we can replace url.Parse with url.ParseRequestURI.
	decodedURL, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	return decodedURL.String(), nil
}

// Rationalize updates the config fields based on the values of other fields.
func Rationalize(c *Config) error {
	// The EnableEmptyManagedFolders flag must be set to true to enforce folder prefixes for Hierarchical buckets.
	if c.EnableHns {
		c.List.EnableEmptyManagedFolders = true
	}

	var err error
	if c.GcsConnection.CustomEndpoint, err = decodeURL(c.GcsConnection.CustomEndpoint); err != nil {
		return err
	}

	if c.GcsAuth.TokenUrl, err = decodeURL(c.GcsAuth.TokenUrl); err != nil {
		return err
	}

	if c.Debug.Fuse || c.Debug.Gcs || c.Debug.LogMutex {
		c.Logging.Severity = "TRACE"
	}

	return nil
}
