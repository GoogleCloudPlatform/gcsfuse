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

	"github.com/googlecloudplatform/gcsfuse/v3/cfg/shared"
	"github.com/spf13/viper"
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

// getMachineType fetches the machine type, checking user-provided configuration
// first (from CLI flags or config file), and falling back to the metadata server.
func getMachineType(v *viper.Viper) (string, error) {
	// Precedence: CLI flag > Config file > Metadata server.
	// 1. Check if the machine-type flag is set by the user (via CLI flag or config file).
	if v.IsSet(machineTypeFlg) {
		if currentMachineType := v.GetString(machineTypeFlg); currentMachineType != "" {
			return currentMachineType, nil
		}
	}
	// 2. Get machine-type from metadata server.
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

func isFlagPresent(flags []string, flag string) bool {
	return slices.Contains(flags, flag)
}

// getOptimizedValue contains the generic logic to determine the optimized value for a flag.
func getOptimizedValue(
	rules *shared.OptimizationRules,
	currentValue any,
	profileName string,
	machineType string,
	input *OptimizationInput,
	machineTypeToGroupMap map[string]string,
	c *Config,
) OptimizationResult {
	// Precedence: Profile -> Machine-type -> Bucket-type -> Default
	// Assuming Machine and Bucket optimizations are applied on mutually exclusive flags, so
	// precedence doesn't matter between them.

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

	// 3. If no profile is set, and no machine-type optimization applies, then check for bucket-type optimization.
	if input != nil && input.BucketType.IsValid() {
		for _, bto := range rules.BucketTypeOptimization {
			for _, bt := range bto.BucketTypes {
				if BucketType(bt) == input.BucketType && matchesConditions(c, bto.Conditions) {
					reason := fmt.Sprintf("bucket-type %q", input.BucketType)
					if len(bto.Conditions) > 0 {
						reason = fmt.Sprintf("bucket-type %q with matching conditions", input.BucketType)
					}
					return OptimizationResult{
						FinalValue:         bto.Value,
						OptimizationReason: reason,
						Optimized:          true,
					}
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

// matchesConditions checks if the current config satisfies all the given conditions.
// If conditions is empty or nil, it returns true.
func matchesConditions(c *Config, conditions map[string]any) bool {
	if len(conditions) == 0 {
		return true
	}
	if c == nil {
		return false
	}
	for path, expectedVal := range conditions {
		actualVal, ok := getConfigValueByPath(c, path)
		if !ok || !valuesMatch(actualVal, expectedVal) {
			return false
		}
	}
	return true
}

// getConfigValueByPath traverses a struct by yaml tags corresponding to the dot-separated path.
func getConfigValueByPath(config any, path string) (any, bool) {
	if config == nil || path == "" {
		return nil, false
	}
	parts := strings.Split(path, ".")
	curr := reflect.ValueOf(config)
	for curr.Kind() == reflect.Pointer || curr.Kind() == reflect.Interface {
		if curr.IsNil() {
			return nil, false
		}
		curr = curr.Elem()
	}

	for _, part := range parts {
		if curr.Kind() != reflect.Struct {
			return nil, false
		}
		found := false
		currType := curr.Type()
		for i := 0; i < curr.NumField(); i++ {
			field := currType.Field(i)
			tag := field.Tag.Get("yaml")
			tagName := strings.Split(tag, ",")[0]
			if tagName == part {
				curr = curr.Field(i)
				for curr.Kind() == reflect.Pointer || curr.Kind() == reflect.Interface {
					if curr.IsNil() {
						return nil, false
					}
					curr = curr.Elem()
				}
				found = true
				break
			}
		}
		if !found {
			return nil, false
		}
	}
	if !curr.CanInterface() {
		return nil, false
	}
	return curr.Interface(), true
}

// setConfigValueByPath sets a field on a struct pointer by traversing yaml tags.
func setConfigValueByPath(config any, path string, val any) error {
	if config == nil || path == "" {
		return fmt.Errorf("invalid config or path")
	}
	curr := reflect.ValueOf(config)
	if curr.Kind() != reflect.Pointer || curr.IsNil() {
		return fmt.Errorf("config must be a non-nil pointer")
	}
	curr = curr.Elem()

	parts := strings.Split(path, ".")
	for i, part := range parts {
		for curr.Kind() == reflect.Pointer || curr.Kind() == reflect.Interface {
			if curr.IsNil() {
				return fmt.Errorf("nil pointer at %s", part)
			}
			curr = curr.Elem()
		}
		if curr.Kind() != reflect.Struct {
			return fmt.Errorf("expected struct at %s", part)
		}
		found := false
		currType := curr.Type()
		for j := 0; j < curr.NumField(); j++ {
			field := currType.Field(j)
			tag := field.Tag.Get("yaml")
			tagName := strings.Split(tag, ",")[0]
			if tagName == part {
				fieldVal := curr.Field(j)
				if i == len(parts)-1 {
					if !fieldVal.CanSet() {
						return fmt.Errorf("cannot set field %s", field.Name)
					}
					valToSet := reflect.ValueOf(val)
					if !valToSet.IsValid() {
						return fmt.Errorf("invalid value to set")
					}
					if valToSet.Type().ConvertibleTo(fieldVal.Type()) {
						fieldVal.Set(valToSet.Convert(fieldVal.Type()))
					} else {
						fieldVal.Set(valToSet)
					}
					return nil
				}
				curr = fieldVal
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("field %s not found in path %s", part, path)
		}
	}
	return nil
}

func isIntKind(k reflect.Kind) bool {
	return k >= reflect.Int && k <= reflect.Int64
}

func isUintKind(k reflect.Kind) bool {
	return k >= reflect.Uint && k <= reflect.Uint64
}

func isFloatKind(k reflect.Kind) bool {
	return k == reflect.Float32 || k == reflect.Float64
}

// valuesMatch compares two values, allowing for type flexibility between integers and floats.
func valuesMatch(actual, expected any) bool {
	if reflect.DeepEqual(actual, expected) {
		return true
	}
	actualVal := reflect.ValueOf(actual)
	expectedVal := reflect.ValueOf(expected)
	if !actualVal.IsValid() || !expectedVal.IsValid() {
		return false
	}

	if actualVal.Kind() == reflect.Bool && expectedVal.Kind() == reflect.Bool {
		return actualVal.Bool() == expectedVal.Bool()
	}
	if actualVal.Kind() == reflect.String && expectedVal.Kind() == reflect.String {
		return actualVal.String() == expectedVal.String()
	}

	if isIntKind(actualVal.Kind()) && isIntKind(expectedVal.Kind()) {
		return actualVal.Int() == expectedVal.Int()
	}
	if isUintKind(actualVal.Kind()) && isUintKind(expectedVal.Kind()) {
		return actualVal.Uint() == expectedVal.Uint()
	}
	if isFloatKind(actualVal.Kind()) && isFloatKind(expectedVal.Kind()) {
		return actualVal.Float() == expectedVal.Float()
	}
	if isIntKind(actualVal.Kind()) && isUintKind(expectedVal.Kind()) {
		return actualVal.Int() >= 0 && uint64(actualVal.Int()) == expectedVal.Uint()
	}
	if isUintKind(actualVal.Kind()) && isIntKind(expectedVal.Kind()) {
		return expectedVal.Int() >= 0 && actualVal.Uint() == uint64(expectedVal.Int())
	}
	if isIntKind(actualVal.Kind()) && isFloatKind(expectedVal.Kind()) {
		return float64(actualVal.Int()) == expectedVal.Float()
	}
	if isFloatKind(actualVal.Kind()) && isIntKind(expectedVal.Kind()) {
		return actualVal.Float() == float64(expectedVal.Int())
	}
	if isUintKind(actualVal.Kind()) && isFloatKind(expectedVal.Kind()) {
		return float64(actualVal.Uint()) == expectedVal.Float()
	}
	if isFloatKind(actualVal.Kind()) && isUintKind(expectedVal.Kind()) {
		return actualVal.Float() == float64(expectedVal.Uint())
	}

	return false
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
