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

package operations_test

import (
	"fmt"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/creds_tests"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/parallel"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
	"github.com/stretchr/testify/suite"
)

func runOperationsSuites(t *testing.T, runCfg parallel.RunConfiguration) {
	mntDir := setup.GetSuiteMntDir(runCfg.MntDir, runCfg.TestBucket, runCfg.MountType)
	runCfg.MntDir = mntDir

	suitesToRun := []suite.TestingSuite{
		&CopyDirSuite{runCfg: runCfg},
		&CopyFileSuite{runCfg: runCfg},
		&CreateThreeLevelDirSuite{runCfg: runCfg},
		&DeleteDirSuite{runCfg: runCfg},
		&DeleteFileSuite{runCfg: runCfg},
		&FileAndDirAttributesSuite{runCfg: runCfg},
		&ListDirSuite{runCfg: runCfg},
		&MoveFileSuite{runCfg: runCfg},
		&ParallelDiropsSuite{runCfg: runCfg},
		&ReadSuite{runCfg: runCfg},
		&RenameDirSuite{runCfg: runCfg},
		&RenameFileSuite{runCfg: runCfg},
		&StatFileSuite{runCfg: runCfg},
		&WriteTestSuite{runCfg: runCfg},
	}

	for _, s := range suitesToRun {
		s := s
		t.Run(fmt.Sprintf("%T", s), func(t *testing.T) {
			t.Parallel()
			suite.Run(t, s)
		})
	}
}

func TestOperations(t *testing.T) {
	t.Parallel()
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	plan := parallel.BuildExecutionPlan(ctx, t, cfg.Operations, nil, nil, "")

	parallel.RunParallelTestDriver(t, ctx, storageClient, plan, runOperationsSuites)
}

func TestOperationsCreds(t *testing.T) {
	t.Parallel()
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	credsAuthModes := []setup.AuthMode{setup.EnvVar, setup.KeyFileOnly, setup.EnvVarAndKeyFile}

	plan := creds_tests.BuildCredsExecutionPlan(
		ctx, t, storageClient, cfg.Operations, "objectAdmin",
		[]setup.MountType{setup.StaticMount}, credsAuthModes,
	)

	parallel.RunParallelTestDriver(t, ctx, storageClient, plan, runOperationsSuites)
}

