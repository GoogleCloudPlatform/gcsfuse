package config

// OverrideWithLoggingFlags overwrites the configs with the flag values if the
// config values are empty.
func OverrideWithLoggingFlags(mountConfig *MountConfig, logFile string, logFormat string,
	debugFuse bool, debugGCS bool, debugMutex bool) {
	// If log file is not set in config file, override it with flag value.
	if mountConfig.LogConfig.File == "" {
		mountConfig.LogConfig.File = logFile
	}
	// If log format is not set in config file, override it with flag value.
	if mountConfig.LogConfig.Format == "" {
		mountConfig.LogConfig.Format = logFormat
	}
	// If debug_fuse, debug_gcsfuse or debug_mutex flag is set, override log
	// severity to TRACE.
	if debugFuse || debugGCS || debugMutex {
		mountConfig.LogConfig.Severity = TRACE
	}
}
