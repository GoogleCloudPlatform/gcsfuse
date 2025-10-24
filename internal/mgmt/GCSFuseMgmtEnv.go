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

package mgmt

import (
	"bufio"
	"os"
	"strings"
)

// HasDockerEnv checks if the /.dockerenv file exists.
func (mgmt *GCSFuseMgmtService) hasDockerEnv() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	return false
}

// CheckCgroup checks /proc/1/cgroup for container-related keywords.
func (mgmt *GCSFuseMgmtService) checkDockerProc() bool {
	file, err := os.Open(dockerProcess)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "docker") ||
			strings.Contains(line, "kubepods") ||
			strings.Contains(line, "containerd") {
			return true
		}
	}
	return false
}

// CheckK8sEnv checks for common Kubernetes environment variables.
func (mgmt *GCSFuseMgmtService) checkK8sEnv() bool {
	return os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}

func (mgmt *GCSFuseMgmtService) detectEnv() {

	mgmt.svc.IsContainer = mgmt.hasDockerEnv() || mgmt.checkDockerProc() || mgmt.checkK8sEnv()

}
