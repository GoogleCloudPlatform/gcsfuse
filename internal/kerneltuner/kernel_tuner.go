package kerneltuner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// KernelParam represents an individual parameter setting.
type KernelParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Scope string `json:"scope"`
}

type KernelParamsConfig struct {
	RequestID  string        `json:"request_id"`
	Timestamp  string        `json:"timestamp"`
	Parameters []KernelParam `json:"parameters"`
}

// KernelParameterBuilder handles the creation of multi kernel parameters.
type KernelParameterBuilder struct {
	params []KernelParam
}

// NewKernelParameters initializes the builder
func NewKernelParameters() *KernelParameterBuilder {
	return &KernelParameterBuilder{
		params: make([]KernelParam, 0),
	}
}

// WithMaxPage sets the fuse-max_pages_limit (Global)
func (b *KernelParameterBuilder) WithMaxPage(limit int) *KernelParameterBuilder {
	b.params = append(b.params, KernelParam{
		Name:  "fuse-max-pages-limit",
		Value: fmt.Sprintf("%d", limit),
		Scope: "global",
	})
	return b
}

// WithTransparentHugePages sets hugepages to madvise (Global)
func (b *KernelParameterBuilder) WithTransparentHugePages(mode string) *KernelParameterBuilder {
	b.params = append(b.params, KernelParam{
		Name:  "transparent-hugepages",
		Value: mode,
		Scope: "global",
	})
	return b
}

// WithReadAheadKb sets the read ahead value (Mount)
func (b *KernelParameterBuilder) WithReadAheadKb(kb int) *KernelParameterBuilder {
	b.params = append(b.params, KernelParam{
		Name:  "read_ahead_kb",
		Value: fmt.Sprintf("%d", kb),
		Scope: "mount",
	})
	return b
}

// WithMaxBackgroundRequests sets background requests (Mount)
func (b *KernelParameterBuilder) WithMaxBackgroundRequests(val int) *KernelParameterBuilder {
	b.params = append(b.params, KernelParam{
		Name:  "fuse-max-background-requests",
		Value: fmt.Sprintf("%d", val),
		Scope: "mount",
	})
	return b
}

// WithCongestionWindowThreshold sets background requests (Mount)
func (b *KernelParameterBuilder) WithCongestionWindowThreshold(val int) *KernelParameterBuilder {
	b.params = append(b.params, KernelParam{
		Name:  "fuse-congestion-window-threshold",
		Value: fmt.Sprintf("%d", val),
		Scope: "mount",
	})
	return b
}

// Apply serializes the current configuration and writes it atomically to the
// target path in case of GKE environment or applies automatically in case of GCE environments
func (b *KernelParameterBuilder) Apply(targetPath string) error {
	config := KernelParamsConfig{
		RequestID:  uuid.New().String(),
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Parameters: b.params,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}

	dir := filepath.Dir(targetPath)
	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Atomic write-rename pattern
	tempFile, err := os.CreateTemp(dir, "kernel-params-tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write(data); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	return os.Rename(tempFile.Name(), targetPath)
}
