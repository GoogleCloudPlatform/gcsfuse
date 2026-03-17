package gcsx

import (
	"os"
	"strconv"
	"strings"
)

// getOSLevelRSS returns the Resident Set Size (RSS) of the current process in bytes.
// It reads from /proc/self/statm on Linux.
func getOSLevelRSS() uint64 {
	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) >= 2 {
		rssPages, err := strconv.ParseUint(fields[1], 10, 64)
		if err == nil {
			return rssPages * uint64(os.Getpagesize())
		}
	}
	return 0
}
