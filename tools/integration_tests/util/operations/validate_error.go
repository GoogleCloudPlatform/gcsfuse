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
