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
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/suite"
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

func cloneRepository() ([]byte, error) {
	return operations.ExecuteToolCommandfInDirectory(testDirPath, tool, "clone %s", repoURL)
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
	output, err := cloneRepository()

	if err != nil {
		s.T().Errorf("Git clone failed: %s: %v", string(output), err)
	}
}

func (s *ignoreInterruptsTest) TestGitCheckout() {
	_, err := cloneRepository()
	if err != nil {
		s.T().Errorf("cloneRepository() failed: %v", err)
	}

	output, err := checkoutBranch(branchName)

	if err != nil {
		s.T().Errorf("Git checkout failed: %s: %v", string(output), err)
	}
}

func (s *ignoreInterruptsTest) TestGitEmptyCommit() {
	_, err := cloneRepository()
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
	_, err := cloneRepository()
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
