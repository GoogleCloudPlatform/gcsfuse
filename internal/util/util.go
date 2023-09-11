package util

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
)

const GCSFUSE_PARENT_PROCESS_DIR = "gcsfuse-parent-process-dir"

// 1. Returns the same filepath in case of absolute path or empty filename.
// 2. For child process, it resolves relative path like, ./test.txt, test.txt
// ../test.txt etc, with respect to GCSFUSE_PARENT_PROCESS_DIR
// because we execute the child process from different directory and input
// files are provided with respect to GCSFUSE_PARENT_PROCESS_DIR.
// 3. For relative path starting with ~, it resolves with respect to home dir.
func GetResolvedPath(filePath string) (resolvedPath string, err error) {
	if filePath == "" || path.IsAbs(filePath) {
		resolvedPath = filePath
		return
	}

	// Relative path starting with tilda (~)
	if strings.HasPrefix(filePath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("fetch home dir: %w", err)
		}
		return filepath.Join(homeDir, filePath[2:]), err
	}

	// We reach here, when relative path starts with . or .. or other than (/ or ~)
	gcsfuseParentProcessDir, _ := os.LookupEnv(GCSFUSE_PARENT_PROCESS_DIR)
	gcsfuseParentProcessDir = strings.TrimSpace(gcsfuseParentProcessDir)
	if gcsfuseParentProcessDir == "" {
		return filepath.Abs(filePath)
	} else {
		return filepath.Join(gcsfuseParentProcessDir, filePath), err
	}
}

func ResolveFilePath(filePath string, configKey string) (resolvedPath string, err error) {
	resolvedPath, err = GetResolvedPath(filePath)
	if filePath == resolvedPath || err != nil {
		return
	}

	logger.Infof("Value of [%s] resolved from [%s] to [%s]\n", configKey, filePath, resolvedPath)
	return resolvedPath, nil
}

// ResolveConfigFilePaths resolved the config file paths specified in the config file.
func ResolveConfigFilePaths(config *config.MountConfig) (err error) {
	config.LogConfig.File, err = ResolveFilePath(config.LogConfig.File, "logging: file")
	if err != nil {
		return
	}
	return
}
