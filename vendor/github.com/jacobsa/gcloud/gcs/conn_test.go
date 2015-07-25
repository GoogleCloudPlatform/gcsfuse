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

// Restrict this test to builds that specify the tag 'integration'.
// +build integration

package gcs_test

import (
	"testing"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"

	"github.com/jacobsa/gcloud/gcs"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestConn(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ConnTest struct {
	ctx  context.Context
	conn gcs.Conn
}

var _ SetUpInterface = &ConnTest{}

func init() { RegisterTestSuite(&ConnTest{}) }

func (t *ConnTest) SetUp(ti *TestInfo) {
	var err error

	t.ctx = ti.Ctx

	// Set up a token source.
	const scope = gcs.Scope_FullControl
	tokenSrc, err := google.DefaultTokenSource(context.Background(), scope)
	AssertEq(nil, err)

	// Use that to create a GCS connection, enabling retry if requested.
	cfg := &gcs.ConnConfig{
		TokenSource: tokenSrc,
	}

	t.conn, err = gcs.NewConn(cfg)
	AssertEq(nil, err)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ConnTest) NonExistentBucket() {
	var err error

	const name = "jklsdfghiouyhiosufhkdjf"
	_, err = t.conn.OpenBucket(t.ctx, name)

	ExpectThat(err, Error(HasSubstr("Unknown bucket")))
	ExpectThat(err, Error(HasSubstr(name)))
}

func (t *ConnTest) BadCredentials() {
	var err error

	const name = "golang"
	_, err = t.conn.OpenBucket(t.ctx, name)

	ExpectThat(err, Error(HasSubstr("Bad credentials")))
	ExpectThat(err, Error(HasSubstr(name)))
}
