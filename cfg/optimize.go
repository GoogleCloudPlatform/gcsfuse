// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

package cfg

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg/shared"
)

////////////////////////////////////////////////////////////////////////
// Constants
////////////////////////////////////////////////////////////////////////

const (
	maxRetries     = 2
	httpTimeout    = 50 * time.Millisecond
	machineTypeFlg = "machine-type"
)

////////////////////////////////////////////////////////////////////////
// Types
////////////////////////////////////////////////////////////////////////

// OptimizationResult holds the outcome of an optimization check, including the
// new value and the reason for the change.
type OptimizationResult struct {
	// FinalValue is the value after applying all the optimizations. This will be the same as the original value if optimizations didn't change anything.
	FinalValue any `yaml:"final_value" json:"final_value"`
	// If value is optimized, then this will contain the description of what optimization caused the change, e.g. "profile aiml-training", or "machine-type a3-highgpu-8g" etc.
	OptimizationReason string `yaml:"optimization_reason" json:"optimization_reason"`
	// Optimized true indicates that the value was changed by optimization (either machine-type based, or profile-based).
	Optimized bool `yaml:"-" json:"-"` // Field hidden from YAML and JSON to avoid it in logs.
}

type isValueSet interface {
	IsSet(string) bool
	GetString(string) string
	GetBool(string) bool
}

// flagOverride represents a flag override with its new value.
type flagOverride struct {
	newValue any
}

////////////////////////////////////////////////////////////////////////
// Variables
////////////////////////////////////////////////////////////////////////

var (
	// metadataEndpoints are the endpoints to try for fetching metadata.
	// Use an array to make provision for https endpoint in the future: https://cloud.google.com/compute/docs/metadata/querying-metadata#metadata_server_endpoints
	metadataEndpoints = []string{
		"http://metadata.google.internal/computeMetadata/v1/instance/machine-type",
	}
)

////////////////////////////////////////////////////////////////////////
// Helper Functions
////////////////////////////////////////////////////////////////////////

// getMetadata fetches metadata from a given endpoint.
func getMetadata(client *http.Client, endpoint string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", endpoint, err)
	}
	req.Header.Add("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request to %s returned non-OK status: %d", endpoint, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from %s: %w", endpoint, err)
	}

	return body, nil
}

// getMachineType fetches the machine type from the metadata server if not set in isSet, cfg.
func getMachineType(isSet isValueSet, cfg *Config) (string, error) {
	// Precedence: 1. CLI flag, 2. Config file, 3. Metadata server.
	// 1. Check if the machine-type flag is set in CLI flag.
	if isSet.IsSet(machineTypeFlg) {
		if currentMachineType := isSet.GetString(machineTypeFlg); currentMachineType != "" {
			return currentMachineType, nil
		}
	}
	// 2. Check if machine-type flag is set in config-file.
	if cfg != nil && cfg.MachineType != "" {
		return cfg.MachineType, nil
	}
	// 3. Get machine-type from metadata server.
	client := http.Client{Timeout: httpTimeout}
	for range maxRetries {
		for _, endpoint := range metadataEndpoints {
			body, err := getMetadata(&client, endpoint)
			if err != nil {
				continue
			}

			currentMachineType := string(body)
			parts := strings.Split(currentMachineType, "/")
			return parts[len(parts)-1], nil
		}
	}

	return "", fmt.Errorf("failed to get machine type from any metadata endpoint after retries")
}

// convertToCamelCase converts a string from snake-case to CamelCase.
func convertToCamelCase(input string) string {
	if input == "" {
		return ""
	}

	// Split the string by hyphen.
	parts := strings.Split(input, "-")

	// Capitalize each part and join them together.
	for i, part := range parts {
		if len(part) > 0 {
			runes := []rune(part)
			runes[0] = unicode.ToUpper(runes[0])
			parts[i] = string(runes)
		}
	}

	return strings.Join(parts, "")
}

// setFlagValue uses reflection to set the value of a flag in ServerConfig.
func setFlagValue(cfg *Config, flag string, override flagOverride, isSet isValueSet) error {
	// Split the flag name into parts to traverse nested structs.
	parts := strings.Split(flag, ".")
	if len(parts) == 0 {
		return fmt.Errorf("invalid flag name: %s", flag)
	}

	// Start with the Config.
	v := reflect.ValueOf(cfg).Elem()
	var field reflect.Value
	// Traverse nested structs.
	for _, part := range parts {
		field = v.FieldByName(convertToCamelCase(part))
		if !field.IsValid() {
			return fmt.Errorf("invalid flag name: %s", flag)
		}
		v = field
	}

	// Check if the field exists.
	if !field.IsValid() {
		return fmt.Errorf("invalid flag name: %s", flag)
	}

	// Check if the field is settable.
	if !field.CanSet() {
		return fmt.Errorf("cannot set flag: %s", flag)
	}

	// Construct the full flag name for IsSet check.
	fullFlagName := strings.ToLower(flag)

	// Only override if the user hasn't set it.
	if !isSet.IsSet(fullFlagName) {
		// Set the value based on the field type.

		switch field.Kind() {
		case reflect.Bool:
			boolValue, ok := override.newValue.(bool)
			if !ok {
				return fmt.Errorf("invalid boolean value for flag %s: %v", flag, override.newValue)
			}
			field.SetBool(boolValue)
		case reflect.Int, reflect.Int64:
			intValue, ok := override.newValue.(int)
			if !ok {
				return fmt.Errorf("invalid integer value for flag %s: %v", flag, override.newValue)
			}
			field.SetInt(int64(intValue))
		case reflect.String:
			stringValue, ok := override.newValue.(string)
			if !ok {
				return fmt.Errorf("invalid string value for flag %s: %v", flag, override.newValue)
			}
			field.SetString(stringValue)
		default:
			return fmt.Errorf("unsupported flag type for flag %s", flag)
		}
	}

	return nil
}

func isFlagPresent(flags []string, flag string) bool {
	return slices.Contains(flags, flag)
}

// getOptimizedValue contains the generic logic to determine the optimized value for a flag.
func getOptimizedValue(
	rules *shared.OptimizationRules,
	currentValue any,
	profileName string,
	machineType string,
	machineTypeToGroupMap map[string]string,
	bucketType string,
) OptimizationResult {
	// Precedence: Profile -> Machine -> BucketType -> Default

	// 1. If a profile with the given name is active and has optimization defined for it, then it takes precedence.
	for _, p := range rules.Profiles {
		if p.Name == profileName {
			return OptimizationResult{
				FinalValue:         p.Value,
				OptimizationReason: fmt.Sprintf("profile %q", profileName),
				Optimized:          true,
			}
		}
	}

	// 2. Only if no profile is set, check for a machine-based optimization.
	if group, ok := machineTypeToGroupMap[machineType]; ok {
		for _, mbo := range rules.MachineBasedOptimization {
			if mbo.Group == group {
				return OptimizationResult{
					FinalValue:         mbo.Value,
					OptimizationReason: fmt.Sprintf("machine-type group %q", group),
					Optimized:          true,
				}
			}
		}
	}

	// 3. Only if no profile or machine-based optimization is found, check for bucket-type-based optimization.
	if bucketType != "" {
		for _, bto := range rules.BucketTypeOptimization {
			if bto.BucketType == bucketType {
				return OptimizationResult{
					FinalValue:         bto.Value,
					OptimizationReason: fmt.Sprintf("bucket-type %q", bucketType),
					Optimized:          true,
				}
			}
		}
	}

	// 4. If no optimization is found, return the original value.
	return OptimizationResult{
		FinalValue: currentValue,
		Optimized:  false,
	}
}

// CreateHierarchicalOptimizedFlags converts a flat map with dot-separated keys
// into a nested map structure.
// It returns an error if a key prefix conflict is detected.
func CreateHierarchicalOptimizedFlags(flatMap map[string]OptimizationResult) (map[string]any, error) {
	nestedMap := make(map[string]any)

	for key, value := range flatMap {
		parts := strings.Split(key, ".")
		currentLevel := nestedMap

		// Traverse the path and create intermediate maps.
		for i, part := range parts[:len(parts)-1] {
			// Intermediate part, ensure the next level map exists
			if existingVal, exists := currentLevel[part]; exists {
				if _, isMap := existingVal.(map[string]any); !isMap {
					return nil, fmt.Errorf("key conflict: %q is both a path and a terminal key", strings.Join(parts[0:i+1], "."))
				}
				currentLevel = existingVal.(map[string]any)
			} else {
				newLevel := make(map[string]any)
				currentLevel[part] = newLevel
				currentLevel = newLevel
			}
		}

		// Set the value at the final key.
		lastKey := parts[len(parts)-1]
		if _, exists := currentLevel[lastKey]; exists {
			return nil, fmt.Errorf("key conflict: %q is both a path and a terminal key", key)
		}
		currentLevel[lastKey] = value
	}
	return nestedMap, nil
}

// ApplyBucketTypeOptimizations applies bucket-type-based optimizations to the config.
// This should be called after the bucket type is known (i.e., at mount time).
// bucketType should be one of: "hierarchical", "zonal", "standard"
func (c *Config) ApplyBucketTypeOptimizations(bucketType string, userSetFlags []string) map[string]OptimizationResult {
	optimizedFlags := make(map[string]OptimizationResult)

	// Skip if autoconfig is disabled or if profile is set (profile takes precedence)
	if c.DisableAutoconfig || c.Profile != "" {
		return optimizedFlags
	}

	// Helper function to check if a flag was set by the user
	isFlagSetByUser := func(flagPath string) bool {
		return slices.Contains(userSetFlags, flagPath)
	}

	// Apply optimizations for FUSE settings based on bucket type
	flagsToOptimize := []struct {
		flagPath   string
		currentVal any
		setterFunc func(any) error
	}{
		{
			flagPath:   "file-system.max-background",
			currentVal: c.FileSystem.MaxBackground,
			setterFunc: func(val any) error {
				if v, ok := val.(int64); ok {
					c.FileSystem.MaxBackground = v
					return nil
				}
				return fmt.Errorf("invalid type for max-background")
			},
		},
		{
			flagPath:   "file-system.congestion-threshold",
			currentVal: c.FileSystem.CongestionThreshold,
			setterFunc: func(val any) error {
				if v, ok := val.(int64); ok {
					c.FileSystem.CongestionThreshold = v
					return nil
				}
				return fmt.Errorf("invalid type for congestion-threshold")
			},
		},
		{
			flagPath:   "file-system.async-read",
			currentVal: c.FileSystem.AsyncRead,
			setterFunc: func(val any) error {
				if v, ok := val.(bool); ok {
					c.FileSystem.AsyncRead = v
					return nil
				}
				return fmt.Errorf("invalid type for async-read")
			},
		},
	}

	// Apply each optimization
	for _, flagOpt := range flagsToOptimize {
		// Skip if user explicitly set this flag
		if isFlagSetByUser(flagOpt.flagPath) {
			continue
		}

		// Get optimization rules for this flag
		if rules, ok := AllFlagOptimizationRules[flagOpt.flagPath]; ok {
			result := getOptimizedValue(&rules, flagOpt.currentVal, "", "", nil, bucketType)
			if result.Optimized {
				if err := flagOpt.setterFunc(result.FinalValue); err == nil {
					optimizedFlags[flagOpt.flagPath] = result
				}
			}
		}
	}

	return optimizedFlags
}

// GetBucketTypeString converts bucket type flags to a string representation
// Priority: zonal > hierarchical > standard
// This ensures zonal buckets get FUSE optimizations even if they are also HNS-enabled
func GetBucketTypeString(hierarchical bool, zonal bool) string {
	if zonal {
		return "zonal"
	}
	if hierarchical {
		return "hierarchical"
	}
	return "standard"
}
