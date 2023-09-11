package main

import "github.com/googlecloudplatform/gcsfuse/internal/config"

func overrideWithLoggingFlags(mountConfig *config.MountConfig, flags *flagStorage) {
	// if log file is not set in config file, override it with flag value.
	if mountConfig.LogConfig.File == "" {
		mountConfig.LogConfig.File = flags.LogFile
	}
	// if log format is not set in config file, override it with flag value.
	if mountConfig.LogConfig.Format == "" {
		mountConfig.LogConfig.Format = flags.LogFormat
	}
	// if debug_fuse, debug_gcsfuse or debug_mutex flag is set, override log
	// severity to TRACE.
	if flags.DebugFuse || flags.DebugGCS || flags.DebugMutex {
		mountConfig.LogConfig.Severity = config.TRACE
	}
}

func resolveConfigFilePaths(config *config.MountConfig) (err error) {
	config.LogConfig.File, err = resolveFilePath(config.LogConfig.File, "logging: file")
	if err != nil {
		return
	}
	return
}
