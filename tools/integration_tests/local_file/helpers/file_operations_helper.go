package helpers

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

func VerifyRenameOperationNotSupported(err error, t *testing.T) {
	if err == nil || !strings.Contains(err.Error(), "operation not supported") {
		t.Fatalf("os.Rename(), expected err: %s, got err: %v",
			"operation not supported", err)
	}
}
