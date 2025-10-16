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
