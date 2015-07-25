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

package gcsutil

import (
	"fmt"
	"io/ioutil"

	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

// Read the contents of the latest generation of the object with the supplied
// name.
func ReadObject(
	ctx context.Context,
	bucket gcs.Bucket,
	name string) (contents []byte, err error) {
	// Call the bucket.
	req := &gcs.ReadObjectRequest{
		Name: name,
	}

	rc, err := bucket.NewReader(ctx, req)
	if err != nil {
		return
	}

	// Don't forget to close.
	defer func() {
		closeErr := rc.Close()
		if closeErr != nil && err == nil {
			err = fmt.Errorf("Close: %v", closeErr)
		}
	}()

	// Read the contents.
	contents, err = ioutil.ReadAll(rc)
	if err != nil {
		err = fmt.Errorf("ReadAll: %v", err)
		return
	}

	return
}
