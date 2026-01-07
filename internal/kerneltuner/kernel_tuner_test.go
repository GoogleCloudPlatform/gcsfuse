package kerneltuner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewKernelParameters(t *testing.T) {
	b := NewKernelParameters()
	if b == nil {
		t.Fatal("NewKernelParameters returned nil")
	}
	if len(b.params) != 0 {
		t.Errorf("Expected empty params, got %d", len(b.params))
	}
}

func TestKernelParameterBuilder_Methods(t *testing.T) {
	tests := []struct {
		name     string
		build    func(*KernelParameterBuilder)
		expected KernelParam
	}{
		{
			name: "WithMaxPage",
			build: func(b *KernelParameterBuilder) {
				b.WithMaxPage(100)
			},
			expected: KernelParam{Name: "fuse-max-pages-limit", Value: "100", Scope: "global"},
		},
		{
			name: "WithTransparentHugePages",
			build: func(b *KernelParameterBuilder) {
				b.WithTransparentHugePages("always")
			},
			expected: KernelParam{Name: "transparent-hugepages", Value: "always", Scope: "global"},
		},
		{
			name: "WithReadAheadKb",
			build: func(b *KernelParameterBuilder) {
				b.WithReadAheadKb(1024)
			},
			expected: KernelParam{Name: "read_ahead_kb", Value: "1024", Scope: "mount"},
		},
		{
			name: "WithMaxBackgroundRequests",
			build: func(b *KernelParameterBuilder) {
				b.WithMaxBackgroundRequests(20)
			},
			expected: KernelParam{Name: "fuse-max-background-requests", Value: "20", Scope: "mount"},
		},
		{
			name: "WithCongestionWindowThreshold",
			build: func(b *KernelParameterBuilder) {
				b.WithCongestionWindowThreshold(15)
			},
			expected: KernelParam{Name: "fuse-congestion-window-threshold", Value: "15", Scope: "mount"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := NewKernelParameters()
			tc.build(b)
			if len(b.params) != 1 {
				t.Fatalf("Expected 1 param, got %d", len(b.params))
			}
			if b.params[0] != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, b.params[0])
			}
		})
	}
}

func TestKernelParameterBuilder_Apply(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kerneltuner_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test creating in a subdirectory to ensure MkdirAll works
	targetPath := filepath.Join(tmpDir, "subdir", "params.json")
	b := NewKernelParameters().WithMaxPage(123)

	err = b.Apply(targetPath)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify file content
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read target file: %v", err)
	}

	var config KernelParamsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if config.RequestID == "" {
		t.Error("RequestID is empty")
	}
	if config.Timestamp == "" {
		t.Error("Timestamp is empty")
	}
	if len(config.Parameters) != 1 {
		t.Fatalf("Expected 1 parameter, got %d", len(config.Parameters))
	}
	expected := KernelParam{Name: "fuse-max-pages-limit", Value: "123", Scope: "global"}
	if config.Parameters[0] != expected {
		t.Errorf("Expected param %v, got %v", expected, config.Parameters[0])
	}
}
