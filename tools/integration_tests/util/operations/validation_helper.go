package operations

import (
	"os"
	"strings"
	"testing"
)

func ValidateNoFileOrDirError(path string, t *testing.T) {
	_, err := os.Stat(path)
	if err == nil || !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("os.Stat(%s). Expected: %s, Got: %v", path,
			"no such file or directory", err)
	}
}

func CheckErrorForReadOnlyFileSystem(err error, t *testing.T) {
	if strings.Contains(err.Error(), "read-only file system") || strings.Contains(err.Error(), "permission denied") || strings.Contains(err.Error(), "Permission denied") {
		return
	}
	t.Errorf("Incorrect error for readonly file system: %v", err.Error())
}
