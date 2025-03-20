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
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/stretchr/testify/suite"
	"testing"
)

type APSizeProviderTestSuite struct {
	suite.Suite
	provider ReadSizeProvider
}

func TestAPSizeProviderTestSuite(t *testing.T) {
	suite.Run(t, new(APSizeProviderTestSuite))
}


func (s *APSizeProviderTestSuite) SetupTest() {
	s.provider = NewAPSizeProvider(1000 * MB, 10)
}

func (s *APSizeProviderTestSuite) TestGetNextReadSize_SequentialRead() {
	s.provider.ProvideFeedback(&Feedback{TotalReadBytes: 1 * MB, ReadCompletely: true, LastOffsetRead: 1 * MB})

	// First read from offset 0
	size, err := s.provider.GetNextReadSize(0)
	s.Require().NoError(err)
	s.Equal(int64(1*MB), size)
	s.Equal(util.Random, s.provider.ReadType())

	// Assumes read all the content.
	s.provider.ProvideFeedback(&Feedback{TotalReadBytes: 1 * MB, ReadCompletely: true, LastOffsetRead: 1 * MB - 1})

	// 2nd reader request from offset 1MiB
	size, err = s.provider.GetNextReadSize(1 * MB)
	s.Require().NoError(err)
	s.Equal(int64(10*MB), size)
	s.Equal(util.Sequential, s.provider.ReadType())

	// Assumes read all the content.
	s.provider.ProvideFeedback(&Feedback{TotalReadBytes: 10 * MB, ReadCompletely: true, LastOffsetRead: 11 * MB - 1})
	
	// 3rd reader request from offset 11MiB
	size, err = s.provider.GetNextReadSize(11 * MB)
	s.Require().NoError(err)
	s.Equal(int64(100*MB), size)
	s.Equal(util.Sequential, s.provider.ReadType())

	// Assumes read all the content.
	s.provider.ProvideFeedback(&Feedback{TotalReadBytes: 100 * MB, ReadCompletely: true, LastOffsetRead: 111 * MB - 1})

	// 4th reader request from offset 111 MiB
	size, err = s.provider.GetNextReadSize(111 * MB)
	s.Require().NoError(err)	
	s.Require().NoError(err)
	s.Equal(int64(889*MB), size)
	s.Equal(util.Sequential, s.provider.ReadType())
}

func (s *APSizeProviderTestSuite) TestGetNextReadSize_RandomRead() {
	size, err := s.provider.GetNextReadSize(5 * MB)
	s.Require().NoError(err)
	s.Equal(int64(1*MB), size)
	s.Equal(util.Random, s.provider.ReadType())

	// Assumes read all the contents.
	s.provider.ProvideFeedback(&Feedback{TotalReadBytes: 1 * MB, ReadCompletely: true, LastOffsetRead: 6 * MB - 1})

	// 2nd random request from offset 11 MiB
	size, err = s.provider.GetNextReadSize(11 * MB)
	s.Require().NoError(err)
	s.Equal(int64(1*MB), size)
	s.Equal(util.Random, s.provider.ReadType())

	// Assuming read all the contents.
	s.provider.ProvideFeedback(&Feedback{TotalReadBytes: 1 * MB, ReadCompletely: true, LastOffsetRead: 12 * MB - 1})

	// 3rd random request from offset 96 MiB
	size, err = s.provider.GetNextReadSize(96 * MB)
	s.Require().NoError(err)
	s.Equal(int64(1*MB), size)
	s.Equal(util.Random, s.provider.ReadType())
}

func (s *APSizeProviderTestSuite) TestGetNextReadSize_InvalidOffset() {
	s.provider.ProvideFeedback(&Feedback{TotalReadBytes: 0, ReadCompletely: false, LastOffsetRead: 0})
	_, err := s.provider.GetNextReadSize(-1)
	s.Require().Error(err)
}

func (s *APSizeProviderTestSuite) TestGetNextReadSize_InvalidOffsetGreaterThanObjectSize() {
	s.provider.ProvideFeedback(&Feedback{TotalReadBytes: 0, ReadCompletely: false, LastOffsetRead: 0})
	_, err := s.provider.GetNextReadSize(1001 * MB)
	s.Require().Error(err)
}

