// Copyright 2025 Google LLC
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

package gcsx

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/stretchr/testify/suite"
)

type RRSizeProvider struct {
	suite.Suite
	provider ReadSizeProvider
}

func TestRRSizeProvider(t *testing.T) {
	suite.Run(t, new(RRSizeProvider))
}

func (s *RRSizeProvider) SetupTest() {
	s.provider = NewRandomReaderReadSizeProvider(1000 * MB)
}

func (s *RRSizeProvider) TestGetNextReadSize_SequentialRead() {
	// First read from offset 0
	size, err := s.provider.GetNextReadSize(0)
	s.Require().NoError(err)
	s.Equal(int64(200*MB), size)
	s.Equal(util.Sequential, s.provider.GetReadType())

	// Assumes read all the content.
	s.provider.ProvideFeedback(&ReadFeedback{TotalBytesRead: 200 * MB, ReadComplete: true, LastOffset: 200*MB - 1})

	// 2nd reader request from offset 1MiB
	size, err = s.provider.GetNextReadSize(200 * MB)
	s.Require().NoError(err)
	s.Equal(int64(200*MB), size)
	s.Equal(util.Sequential, s.provider.GetReadType())

	// Assumes read all the content.
	s.provider.ProvideFeedback(&ReadFeedback{TotalBytesRead: 200 * MB, ReadComplete: true, LastOffset: 400*MB - 1})

	// 3rd reader request from offset 11MiB
	size, err = s.provider.GetNextReadSize(400 * MB)
	s.Require().NoError(err)
	s.Equal(int64(200*MB), size)
	s.Equal(util.Sequential, s.provider.GetReadType())

	// Assumes read all the content.
	s.provider.ProvideFeedback(&ReadFeedback{TotalBytesRead: 200 * MB, ReadComplete: true, LastOffset: 600*MB - 1})

	// 4th reader request from offset 111 MiB
	size, err = s.provider.GetNextReadSize(600 * MB)
	s.Require().NoError(err)
	s.Require().NoError(err)
	s.Equal(int64(200*MB), size)
	s.Equal(util.Sequential, s.provider.GetReadType())

	// Assumes read all the content.
	s.provider.ProvideFeedback(&ReadFeedback{TotalBytesRead: 200 * MB, ReadComplete: true, LastOffset: 800*MB - 1})

	// 4th reader request from offset 111 MiB
	size, err = s.provider.GetNextReadSize(800 * MB)
	s.Require().NoError(err)
	s.Require().NoError(err)
	s.Equal(int64(200*MB), size)
	s.Equal(util.Sequential, s.provider.GetReadType())

	// Assumes read all the content.
	s.provider.ProvideFeedback(&ReadFeedback{TotalBytesRead: 200 * MB, ReadComplete: true, LastOffset: 1000*MB - 1})

	// 4th reader request from offset 111 MiB
	_, err = s.provider.GetNextReadSize(1000 * MB)
	s.Require().Error(err)
}

func (s *RRSizeProvider) TestGetNextReadSize_RandomRead() {
	size, err := s.provider.GetNextReadSize(5 * MB)
	s.Require().NoError(err)
	s.Equal(int64(200*MB), size)
	s.Equal(util.Sequential, s.provider.GetReadType())

	// Assumes read all the contents.
	s.provider.ProvideFeedback(&ReadFeedback{TotalBytesRead: 1 * MB, ReadComplete: false, LastOffset: 6*MB - 1})

	// 2nd random request from offset 11 MiB
	size, err = s.provider.GetNextReadSize(11 * MB)
	s.Require().NoError(err)
	s.Equal(int64(200*MB), size)
	s.Equal(util.Sequential, s.provider.GetReadType())

	// Assuming read all the contents.
	s.provider.ProvideFeedback(&ReadFeedback{TotalBytesRead: 1 * MB, ReadComplete: false, LastOffset: 12*MB - 1})

	// 3rd random request from offset 96 MiB
	size, err = s.provider.GetNextReadSize(96 * MB)
	s.Require().NoError(err)
	s.Equal(int64(1*MB), size)
	s.Equal(util.Random, s.provider.GetReadType())

	s.provider.ProvideFeedback(&ReadFeedback{TotalBytesRead: 1 * MB, ReadComplete: true, LastOffset: 96*MB - 1})

	// 3rd random request from offset 122 MiB
	size, err = s.provider.GetNextReadSize(122 * MB)
	s.Require().NoError(err)
	s.Equal(int64(1*MB), size)
	s.Equal(util.Random, s.provider.GetReadType())
}

func (s *RRSizeProvider) TestGetNextReadSize_InvalidOffset() {
	s.provider.ProvideFeedback(&ReadFeedback{TotalBytesRead: 0, ReadComplete: false, LastOffset: 0})
	_, err := s.provider.GetNextReadSize(-1)
	s.Require().Error(err)
}

func (s *RRSizeProvider) TestGetNextReadSize_InvalidOffsetGreaterThanObjectSize() {
	s.provider.ProvideFeedback(&ReadFeedback{TotalBytesRead: 0, ReadComplete: false, LastOffset: 0})
	_, err := s.provider.GetNextReadSize(1001 * MB)
	s.Require().Error(err)
}
