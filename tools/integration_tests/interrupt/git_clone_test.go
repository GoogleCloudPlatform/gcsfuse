// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Provides integration tests for running git operations with ignore interrupts
// flag/config set.

package interrupt

import (
	"log"
	"math/rand"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/internal/cache/util"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

const (
	repoURL      = "https://github.com/gcsfuse-github-machine-user-bot/test-repository.git"
	repoName     = "test-repository"
	branchName   = "test-branch"
	testFileName = "testFile"
	tool         = "git"
)

var (
	testDirPath string
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ignoreInterruptsTest struct {
	suite.Suite
}

func (s *ignoreInterruptsTest) TearDownTest() {}

func (s *ignoreInterruptsTest) SetupTest() {
	testDirPath = setup.SetupTestDirectory(testDirName)
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (s *ignoreInterruptsTest) cloneRepository() (output []byte, err error) {
	maxAttempts := 5
	isRetryableError := func(err error) bool {
		lowerErr := strings.ToLower(err.Error())
		return strings.Contains(lowerErr, "could not resolve host") || strings.Contains(lowerErr, "could not read from remote repository") || strings.Contains(lowerErr, "failed to connect to github.com")
	}
	for i := range maxAttempts {
		output, err = operations.ExecuteToolCommandfInDirectory(testDirPath, tool, "clone %s", repoURL)

		if err == nil || !isRetryableError(err) {
			return
		}
		s.T().Logf("failed to clone %q with stdout = %q and retryable error = %v", repoURL, string(output), err)
		if i < maxAttempts-1 {
			// Wait for [1ms, 2000ms] before trying again.
			time.Sleep(time.Millisecond * time.Duration(1+rand.Intn(2000)))
		}
	}
	// All retries failed
	return
}

func checkoutBranch(branchName string) ([]byte, error) {
	repositoryPath := path.Join(testDirPath, repoName)
	return operations.ExecuteToolCommandfInDirectory(repositoryPath, tool, "checkout %s", branchName)
}

func emptyCommit() ([]byte, error) {
	repositoryPath := path.Join(testDirPath, repoName)
	return operations.ExecuteToolCommandfInDirectory(repositoryPath, tool, "commit --allow-empty -m \" empty commit\"")
}

func gitAdd(filePath string) ([]byte, error) {
	repositoryPath := path.Join(testDirPath, repoName)
	return operations.ExecuteToolCommandfInDirectory(repositoryPath, tool, "add %s", filePath)
}

func nonEmptyCommit() ([]byte, error) {
	repositoryPath := path.Join(testDirPath, repoName)
	return operations.ExecuteToolCommandfInDirectory(repositoryPath, tool, "commit -m \"test\"")
}

func setGithubUserConfig() {
	repositoryPath := path.Join(testDirPath, repoName)
	output, err := operations.ExecuteToolCommandfInDirectory(repositoryPath, tool, "config user.email \"abc@def.com\"")
	if err != nil {
		log.Printf("Error setting git user.email: %s: %v", string(output), err)
	}
	output, err = operations.ExecuteToolCommandfInDirectory(repositoryPath, tool, "config user.name \"abc\"")
	if err != nil {
		log.Printf("Error setting git user.name: %s: %v", string(output), err)
	}
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *ignoreInterruptsTest) TestGitClone() {
	output, err := s.cloneRepository()

	if err != nil {
		s.T().Errorf("Git clone failed: %s: %v", string(output), err)
	}
}

func (s *ignoreInterruptsTest) TestGitCheckout() {
	_, err := s.cloneRepository()
	if err != nil {
		s.T().Errorf("cloneRepository() failed: %v", err)
	}

	output, err := checkoutBranch(branchName)

	if err != nil {
		s.T().Errorf("Git checkout failed: %s: %v", string(output), err)
	}
}

func (s *ignoreInterruptsTest) TestGitEmptyCommit() {
	_, err := s.cloneRepository()
	if err != nil {
		s.T().Errorf("cloneRepository() failed: %v", err)
	}
	setGithubUserConfig()

	output, err := emptyCommit()

	if err != nil {
		s.T().Errorf("Git empty commit failed: %s: %v", string(output), err)
	}
}

func (s *ignoreInterruptsTest) TestGitCommitWithChanges() {
	_, err := s.cloneRepository()
	if err != nil {
		s.T().Errorf("cloneRepository() failed: %v", err)
	}
	setGithubUserConfig()

	filePath := path.Join(testDirPath, repoName, testFileName)
	operations.CreateFileOfSize(util.MiB, filePath, s.T())
	output, err := gitAdd(filePath)
	if err != nil {
		s.T().Errorf("Git add failed: %s: %v", string(output), err)
	}
	output, err = nonEmptyCommit()

	if err != nil {
		s.T().Errorf("Git commit failed: %s: %v", string(output), err)
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestIgnoreInterrupts(t *testing.T) {
	ts := &ignoreInterruptsTest{}
	suite.Run(t, ts)
}
