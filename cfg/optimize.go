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

type isValueSet interface {
	IsSet(string) bool
	GetString(string) string
	GetBool(string) bool
}

// flagOverride represents a flag override with its new value.
type flagOverride struct {
	newValue interface{}
}

// flagOverrideSet represents a named set of flag overrides.
type flagOverrideSet struct {
	name      string
	overrides map[string]flagOverride
}

// machineType represents a specific machine type with associated flag overrides.
type machineType struct {
	names               []string
	flagOverrideSetName string
}

// optimizationConfig holds the configuration for machine-specific optimizations.
type optimizationConfig struct {
	flagOverrideSets []flagOverrideSet
	machineTypes     []machineType
}

////////////////////////////////////////////////////////////////////////
// Variables
////////////////////////////////////////////////////////////////////////

var (
	// defaultOptimizationConfig provides a default configuration for optimizations.
	defaultOptimizationConfig = optimizationConfig{
		flagOverrideSets: []flagOverrideSet{
			{
				name: "high-performance",
				overrides: map[string]flagOverride{
					"write.global-max-blocks":               {newValue: 1600},
					"metadata-cache.negative-ttl-secs":      {newValue: 0},
					"metadata-cache.ttl-secs":               {newValue: -1},
					"metadata-cache.stat-cache-max-size-mb": {newValue: 1024},
					"metadata-cache.type-cache-max-size-mb": {newValue: 128},
					"implicit-dirs":                         {newValue: true},
					"file-system.rename-dir-limit":          {newValue: 200000},
				},
			},
		},
		machineTypes: []machineType{
			{
				names: []string{
					"a2-megagpu-16g", "a2-ultragpu-8g", "a3-edgegpu-8g", "a3-highgpu-8g", "a3-megagpu-8g", "a3-ultragpu-8g", "a4-highgpu-8g-lowmem",
					"ct5l-hightpu-8t", "ct5lp-hightpu-8t", "ct5p-hightpu-4t", "ct5p-hightpu-4t-tpu", "ct6e-standard-4t", "ct6e-standard-4t-tpu", "ct6e-standard-8t", "ct6e-standard-8t-tpu"},
				flagOverrideSetName: "high-performance",
			},
			// Add more machine types here as needed.
		},
	}

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

// getMachineType fetches the machine type from the metadata server.
func getMachineType(isSet isValueSet) (string, error) {
	// Check if the machine-type flag is set and not empty.
	if isSet.IsSet(machineTypeFlg) {
		if currentMachineType := isSet.GetString(machineTypeFlg); currentMachineType != "" {
			return currentMachineType, nil
		}
	}
	client := http.Client{Timeout: httpTimeout}
	for retry := 0; retry < maxRetries; retry++ {
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

func applyMachineTypeOptimizations(config *optimizationConfig, cfg *Config, isSet isValueSet) map[string]interface{} {
	currentMachineType, err := getMachineType(isSet)
	if err != nil {
		return nil // Non-fatal error, continue with default settings.
	}
	optimizedFlags := make(map[string]interface{})

	// Find the matching machine type.
	mtIndex := slices.IndexFunc(config.machineTypes, func(mt machineType) bool {
		return slices.ContainsFunc(mt.names, func(name string) bool {
			return strings.HasPrefix(currentMachineType, name)
		})
	})

	// If no matching machine type is found, return.
	if mtIndex == -1 {
		return optimizedFlags
	}
	mt := &config.machineTypes[mtIndex]

	// Find the corresponding flag override set.
	flgOverrideSetIndex := slices.IndexFunc(config.flagOverrideSets, func(fos flagOverrideSet) bool {
		return fos.name == mt.flagOverrideSetName
	})

	// If no matching flag override set is found, return.
	if flgOverrideSetIndex == -1 {
		return optimizedFlags
	}
	flgOverrideSet := &config.flagOverrideSets[flgOverrideSetIndex]

	// Apply all overrides from the set.
	for flag, override := range flgOverrideSet.overrides {
		change, err := setFlagValue(cfg, flag, override, isSet)
		if err == nil && change != "" {
			// Split the flag name into parts to create a json like structure.
			parts := strings.Split(flag, ".")
			parent := optimizedFlags
			for i := 0; i < len(parts); i++ {
				if i == len(parts)-1 {
					parent[parts[i]] = change
				} else {
					if _, ok := parent[parts[i]]; !ok {
						parent[parts[i]] = make(map[string]interface{})
					}
					parent = parent[parts[i]].(map[string]interface{})
				}
			}
		}
	}
	return optimizedFlags
}

// Optimize applies machine-type specific optimizations.
func Optimize(cfg *Config, isSet isValueSet) map[string]interface{} {
	// Check if disable-autoconfig is set to true.
	if isSet.GetBool("disable-autoconfig") {
		return nil
	}
	optimizedFlags := applyMachineTypeOptimizations(&defaultOptimizationConfig, cfg, isSet)
	return optimizedFlags
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
func setFlagValue(cfg *Config, flag string, override flagOverride, isSet isValueSet) (string, error) {
	// Split the flag name into parts to traverse nested structs.
	parts := strings.Split(flag, ".")
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid flag name: %s", flag)
	}

	// Start with the Config.
	v := reflect.ValueOf(cfg).Elem()
	var field reflect.Value
	// Traverse nested structs.
	for _, part := range parts {
		field = v.FieldByName(convertToCamelCase(part))
		if !field.IsValid() {
			return "", fmt.Errorf("invalid flag name: %s", flag)
		}
		v = field
	}

	// Check if the field exists.
	if !field.IsValid() {
		return "", fmt.Errorf("invalid flag name: %s", flag)
	}

	// Check if the field is settable.
	if !field.CanSet() {
		return "", fmt.Errorf("cannot set flag: %s", flag)
	}

	// Construct the full flag name for IsSet check.
	fullFlagName := strings.ToLower(flag)

	// The isSet.IsSet() check ensures that we only proceed if the flag has not
	// been explicitly set by the user. In this case, oldValue will be the zero
	// value for its type (e.g., 0, false, "", nil).
	if !isSet.IsSet(fullFlagName) {
		oldValue := field.Interface()
		// Set the value based on the field type.

		switch field.Kind() {
		case reflect.Bool:
			boolValue, ok := override.newValue.(bool)
			if !ok {
				return "", fmt.Errorf("invalid boolean value for flag %s: %v", flag, override.newValue)
			}
			field.SetBool(boolValue)
		case reflect.Int, reflect.Int64:
			intValue, ok := override.newValue.(int)
			if !ok {
				return "", fmt.Errorf("invalid integer value for flag %s: %v", flag, override.newValue)
			}
			field.SetInt(int64(intValue))
		case reflect.String:
			stringValue, ok := override.newValue.(string)
			if !ok {
				return "", fmt.Errorf("invalid string value for flag %s: %v", flag, override.newValue)
			}
			field.SetString(stringValue)
		default:
			return "", fmt.Errorf("unsupported flag type for flag %s", flag)
		}
		return fmt.Sprintf("%v --> %v", oldValue, override.newValue), nil
	}

	return "", nil
}
