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
					"write.enable-streaming-writes":         {newValue: true},
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

// getMachineType fetches the machine type from the metadata server.
func getMachineType(isSet isValueSet) (string, error) {
	// Check if the machine-type flag is set and not empty.
	if isSet.IsSet(machineTypeFlg) {
		machineType := isSet.GetString(machineTypeFlg)
		if machineType != "" {
			return machineType, nil
		}
	}
	client := http.Client{Timeout: httpTimeout}
	for retry := 0; retry < maxRetries; retry++ {
		for _, endpoint := range metadataEndpoints {

			req, err := http.NewRequest(http.MethodGet, endpoint, nil)
			if err != nil {
				return "", fmt.Errorf("failed to create request for %s: %w", endpoint, err)
			}
			req.Header.Add("Metadata-Flavor", "Google")

			resp, err := client.Do(req)
			if err != nil {
				continue // Try the next endpoint.
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				continue
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", fmt.Errorf("failed to read response body from %s: %w", endpoint, err)
			}

			machineType := string(body)
			parts := strings.Split(machineType, "/")
			if len(parts) > 0 {
				return parts[len(parts)-1], nil
			}
			return machineType, nil
		}
	}

	return "", fmt.Errorf("failed to get machine type from any metadata endpoint after retries")
}

// applyMachineTypeOptimizations applies optimizations based on the detected machine type.
func applyMachineTypeOptimizations(config *optimizationConfig, cfg *Config, isSet isValueSet) ([]string, error) {
	machineType, err := getMachineType(isSet)
	if err != nil {
		return []string{}, nil // Non-fatal error, continue with default settings.
	}
	var optimizedFlags []string
	for _, mt := range config.machineTypes {
		for _, name := range mt.names {
			if strings.HasPrefix(machineType, name) {
				// Find the FlagOverrideSet.
				var flgOvrrideSet *flagOverrideSet
				for _, flgOverrideSet := range config.flagOverrideSets {
					if flgOverrideSet.name == mt.flagOverrideSetName {
						flgOvrrideSet = &flgOverrideSet
						break
					}
				}

				if flgOvrrideSet == nil {
					continue
				}

				for flag, override := range flgOvrrideSet.overrides {
					// Use reflection to find the field in ServerConfig.
					err := setFlagValue(cfg, flag, override, isSet)
					if err == nil {
						optimizedFlags = append(optimizedFlags, flag)
					}
				}
				return optimizedFlags, nil // Applied optimizations, no need to check other machine types.
			}
		}
	}

	return optimizedFlags, nil
}

// Optimize applies machine type optimizations using the default configuration.
func Optimize(cfg *Config, isSet isValueSet) ([]string, error) {
	// Check if disable-autoconfig is set to true.
	if isSet.GetBool("disable-autoconfig") {
		return []string{}, nil
	}
	optimizedFlags, err := applyMachineTypeOptimizations(&defaultOptimizationConfig, cfg, isSet)
	return optimizedFlags, err
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

	// Start with the ServerConfig.
	v := reflect.ValueOf(cfg).Elem()
	var field reflect.Value
	// Traverse nested structs.
	for i := 0; i < len(parts); i++ {
		field = v.FieldByName(convertToCamelCase(parts[i]))
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
	for _, v := range flags {
		if v == flag {
			return true
		}
	}
	return false
}
