// Copyright 2023 Google LLC
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
	"math"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"
	"unicode"
)

////////////////////////////////////////////////////////////////////////
// Constants
////////////////////////////////////////////////////////////////////////

const (
	maxRetries   = 3
	initialDelay = 100 * time.Millisecond
	httpTimeout  = 5 * time.Second
)

////////////////////////////////////////////////////////////////////////
// Types
////////////////////////////////////////////////////////////////////////

// isSet interface is abstraction over the IsValueSet() method of viper, specially
// added to keep rationalize method simple. IsValueSet will be used to resolve
// conflicting deprecated flags and new configs.
type isValueSet interface {
	IsSet(string) bool
	GetString(string) string
	GetBool(string) bool
}

// FlagOverride represents a flag override with its new value.
type FlagOverride struct {
	NewValue interface{} `json:"newValue"`
}

// FlagOverrideSet represents a named set of flag overrides.
type FlagOverrideSet struct {
	Name      string                  `json:"name"`
	Overrides map[string]FlagOverride `json:"overrides"`
}

// MachineType represents a specific machine type with associated flag overrides.
type MachineType struct {
	Names               []string `json:"names"`
	FlagOverrideSetName string   `json:"flagOverrideSetName"`
}

// OptimizationConfig holds the configuration for machine-specific optimizations.
type OptimizationConfig struct {
	FlagOverrideSets []FlagOverrideSet `json:"flagOverrideSets"`
	MachineTypes     []MachineType     `json:"machineTypes"`
}

////////////////////////////////////////////////////////////////////////
// Variables
////////////////////////////////////////////////////////////////////////

var (
	// DefaultOptimizationConfig provides a default configuration for optimizations.
	DefaultOptimizationConfig = OptimizationConfig{
		FlagOverrideSets: []FlagOverrideSet{
			{
				Name: "high-performance",
				Overrides: map[string]FlagOverride{
					"write.enable-streaming-writes":         {NewValue: true},
					"metadata-cache.negative-ttl-secs":      {NewValue: 0},
					"metadata-cache.ttl-secs":               {NewValue: -1},
					"metadata-cache.stat-cache-max-size-mb": {NewValue: 1024},
					"metadata-cache.type-cache-max-size-mb": {NewValue: 128},
					"implicit-dirs":                         {NewValue: true},
					"file-system.rename-dir-limit":          {NewValue: 200000},
					"file-system.gid":                       {NewValue: 1000},
				},
			},
		},
		MachineTypes: []MachineType{
			{
				Names:               []string{"a3-highgpu-4g", "a3-highgpu-8g", "a3-megagpu-8g", "a3-ultragpu-8g", "a3-ultragpu-8g-nolssd", "a4-highgpu-8g-lowmem", "ct5l-hightpu-8t", "ct5lp-hightpu-8t", "ct5p-hightpu-4t", "ct5p-hightpu-4t-tpu", "ct6e-standard-4t", "ct6e-standard-4t-tpu", "ct6e-standard-8t", "ct6e-standard-8t-tpu"},
				FlagOverrideSetName: "high-performance",
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
	if isSet.IsSet("machine-type") {
		machineType := isSet.GetString("machine-type")
		if machineType != "" {
			return machineType, nil
		}
	}
	client := http.Client{Timeout: httpTimeout}

	for retry := 0; retry <= maxRetries; retry++ {
		for _, endpoint := range metadataEndpoints {
			req, err := http.NewRequest(http.MethodGet, endpoint, nil)
			if err != nil {
				return "", fmt.Errorf("failed to create request for %s: %w", endpoint, err)
			}
			req.Header.Add("Metadata-Flavor", "Google")

			resp, err := client.Do(req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to get machine type from %s: %v\n", endpoint, err)
				continue // Try the next endpoint.
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
				if retry < maxRetries {
					delay := time.Duration(float64(initialDelay) * math.Pow(2, float64(retry)))
					time.Sleep(delay)
					break // Retry the request.
				} else {
					return "", fmt.Errorf("metadata server %s returned quota error: %d, max retries reached", endpoint, resp.StatusCode)
				}
			}

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

// ApplyMachineTypeOptimizations applies optimizations based on the detected machine type.
func ApplyMachineTypeOptimizations(config *OptimizationConfig, cfg *Config, isSet isValueSet) error {
	machineType, err := getMachineType(isSet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to get machine type from metadata server: %v\n", err)
		return nil // Non-fatal error, continue with default settings.
	}

	for _, mt := range config.MachineTypes {
		for _, name := range mt.Names {
			if strings.HasPrefix(machineType, name) {
				// Find the FlagOverrideSet.
				var flagOverrideSet *FlagOverrideSet
				for i := range config.FlagOverrideSets {
					if config.FlagOverrideSets[i].Name == mt.FlagOverrideSetName {
						flagOverrideSet = &config.FlagOverrideSets[i]
						break
					}
				}

				if flagOverrideSet == nil {
					fmt.Fprintf(os.Stderr, "Warning: FlagOverrideSet %s not found for machine type %s.\n", mt.FlagOverrideSetName, mt.Names)
					continue
				}

				for flag, override := range flagOverrideSet.Overrides {
					// Use reflection to find the field in ServerConfig.
					err := setFlagValue(cfg, flag, override, isSet)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Warning: Failed to set flag %s: %v\n", flag, err)
					}
				}
				return nil // Applied optimizations, no need to check other machine types.
			}
		}
	}

	return nil
}

// Optimize applies machine type optimizations using the default configuration.
func Optimize(cfg *Config, isSet isValueSet) error {
	// Check if disable-autoconfig is set to true.
	if isSet.GetBool("disable-autoconfig") {
		return nil
	}
	return ApplyMachineTypeOptimizations(&DefaultOptimizationConfig, cfg, isSet)
}

// snakeCaseToCamelCase converts a string from snake-case to CamelCase.
func convertToCamelCase(input string) string {
	if input == "" {
		return ""
	}

	// Split the string by underscores.
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
func setFlagValue(cfg *Config, flag string, override FlagOverride, isSet isValueSet) error {
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
			boolValue, ok := override.NewValue.(bool)
			if !ok {
				return fmt.Errorf("invalid boolean value for flag %s: %v", flag, override.NewValue)
			}
			field.SetBool(boolValue)
		case reflect.Int, reflect.Int64:
			intValue, ok := override.NewValue.(int)
			if !ok {
				return fmt.Errorf("invalid integer value for flag %s: %v", flag, override.NewValue)
			}
			field.SetInt(int64(intValue))
		case reflect.String:
			stringValue, ok := override.NewValue.(string)
			if !ok {
				return fmt.Errorf("invalid string value for flag %s: %v", flag, override.NewValue)
			}
			field.SetString(stringValue)
		default:
			return fmt.Errorf("unsupported flag type for flag %s", flag)
		}
	}

	return nil
}
