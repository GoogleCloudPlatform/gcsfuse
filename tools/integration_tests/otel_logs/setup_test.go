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

package otel_logs

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
	"github.com/stretchr/testify/suite"
)

var (
	mockServerURL        string
	mockServer           *http.Server
	logRecords           [][]byte
	logMu                sync.Mutex
	configFileForGCSFuse string
)

func startMockServer() error {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return err
	}
	mockServerURL = listener.Addr().String()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		log.Printf("Mock server received request on %s. Content-Type: %s, Body length: %d", r.URL.Path, r.Header.Get("Content-Type"), len(body))

		if r.URL.Path == "/v1/logs" && len(body) > 0 {
			logMu.Lock()
			logRecords = append(logRecords, body)
			logMu.Unlock()
		}
		w.WriteHeader(http.StatusOK)
	})

	mockServer = &http.Server{Handler: mux}
	go func() {
		_ = mockServer.Serve(listener)
	}()
	return nil
}

func stopMockServer() {
	if mockServer != nil {
		_ = mockServer.Close()
	}
}

const (
	testDirName = "otel_logs"
	gkeTempDir  = "/gcsfuse-tmp"
)

type env struct {
	storageClient *storage.Client
	ctx           context.Context
	testDirPath   string
	cfg           *test_suite.TestConfig
	bucketType    string
}

var (
	testEnv   env
	mountFunc func(*test_suite.TestConfig, []string) error
)

type OTelLogsTestBase struct {
	suite.Suite
	flags []string
}

func (p *OTelLogsTestBase) SetupSuite() {
	p.flags = append(p.flags, "--config-file="+configFileForGCSFuse)
	setup.SetUpLogFilePath(p.T().Name(), p.flags, gkeTempDir, "", testEnv.cfg)
	mountGCSFuseAndSetupTestDir(p.flags, testEnv.ctx, testEnv.storageClient)
}

func (p *OTelLogsTestBase) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (p *OTelLogsTestBase) SetupTest() {
	logMu.Lock()
	logRecords = nil
	logMu.Unlock()

	testName := strings.ReplaceAll(p.T().Name(), "/", "_")
	gcsDir := path.Join(testDirName, testName)
	testEnv.testDirPath = client.SetupTestDirectory(testEnv.ctx, testEnv.storageClient, gcsDir)
	client.SetupFileInTestDirectory(testEnv.ctx, testEnv.storageClient, gcsDir, "hello.txt", 10, p.T())
}

func (p *OTelLogsTestBase) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(p.T())
}

func mountGCSFuseAndSetupTestDir(flags []string, ctx context.Context, storageClient *storage.Client) {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, flags, mountFunc)
	if testEnv.cfg.GKEMountedDirectory != "" {
		setup.SetMntDir(testEnv.cfg.GKEMountedDirectory)
	}
	testEnv.testDirPath = client.SetupTestDirectory(ctx, storageClient, testDirName)
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	if err := startMockServer(); err != nil {
		log.Fatalf("Failed to start mock server: %v", err)
	}
	defer stopMockServer()

	// Create a temp config file for gcsfuse
	configFileForGCSFuse = path.Join(os.TempDir(), "otel_config.yaml")
	configContent := fmt.Sprintf(`
logging:
  otel-logging-enabled: true
  otel-logging-endpoint: "localhost:%s"
`, strings.Split(mockServerURL, ":")[1])
	if err := os.WriteFile(configFileForGCSFuse, []byte(configContent), 0644); err != nil {
		log.Fatalf("Failed to write config file: %v", err)
	}
	defer func() { _ = os.Remove(configFileForGCSFuse) }()

	configFile := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(configFile.OtelLogs) == 0 {
		log.Fatal("No configuration found for OtelLogs in config file.")
	}
	testEnv.cfg = &configFile.OtelLogs[0]

	testEnv.ctx = context.Background()
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, testEnv.cfg)

	var err error
	testEnv.storageClient, err = client.CreateStorageClient(testEnv.ctx)
	if err != nil {
		log.Fatalf("client.CreateStorageClient: %v", err)
	}
	defer func() { _ = testEnv.storageClient.Close() }()

	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(testEnv.cfg.GKEMountedDirectory, m))
	}

	setup.SetUpTestDirForTestBucket(testEnv.cfg)
	setup.OverrideFilePathsInFlagSet(testEnv.cfg, setup.TestDir())

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMountingWithConfigFile
	successCode := m.Run()

	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}
