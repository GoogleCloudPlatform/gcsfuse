// Copyright 2024 Google LLC
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
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

type RetryConfig struct {
	// The name of the method to apply retries to (e.g., JsonCreate, JsonStat).
	Method string `yaml:"method"`
	// Retry instruction (e.g., return-503, stall-33s-after-20K).
	RetryInstruction string `yaml:"retryInstruction"`
	// Number of times to retry.
	RetryCount int `yaml:"retryCount"`
	// Number of starting retry attempts to skip.
	SkipCount int `yaml:"skipCount"`
}

type HeaderValidation struct {
	// Header name to validate (e.g., "user-agent", "x-goog-api-client").
	HeaderName string `yaml:"headerName"`
	// Expected value or pattern to find in the header.
	ExpectedPattern string `yaml:"expectedPattern"`
	// Whether to fail requests if header is missing or doesn't match.
	FailOnMismatch bool `yaml:"failOnMismatch"`
}

type Config struct {
	// ProxyType specifies the protocol: "http" or "grpc"
	ProxyType string `yaml:"proxyType"`
	// TargetHost is the address of emulator server to which proxy server interacts.
	TargetHost       string             `yaml:"targetHost"`
	RetryConfig      []RetryConfig      `yaml:"retryConfig"`
	HeaderValidation []HeaderValidation `yaml:"headerValidation"`
}

func printConfig(config Config) {
	log.Println("Proxy Type:", config.ProxyType)
	log.Println("Target Host:", config.TargetHost)
	for _, retry := range config.RetryConfig {
		log.Println("Method:", retry.Method)
		log.Println("Retry instructions:", retry.RetryInstruction)
		log.Println("Retry Count:", retry.RetryCount)
		log.Println("Skip Count:", retry.SkipCount)
	}
	for _, header := range config.HeaderValidation {
		log.Println("Header Validation - Name:", header.HeaderName)
		log.Println("Header Validation - Expected Pattern:", header.ExpectedPattern)
		log.Println("Header Validation - Fail On Mismatch:", header.FailOnMismatch)
	}
}

func parseConfigFile(configPath string) (*Config, error) {
	var config Config

	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file, %s", err)
	}

	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode into struct, %v", err)
	}

	return &config, nil
}
