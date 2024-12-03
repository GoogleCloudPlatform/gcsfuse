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

package proxy_server

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

type Config struct {
	// TargetHost is the address of emulator server to which proxy server interacts.
	TargetHost  string        `yaml:"targetHost"`
	RetryConfig []RetryConfig `yaml:"retryConfig"`
}

func printConfig(config Config) {
	log.Println("Target Host:", config.TargetHost)
	for _, retry := range config.RetryConfig {
		log.Println("Method:", retry.Method)
		log.Println("Retry instructions:", retry.RetryInstruction)
		log.Println("Retry Count:", retry.RetryCount)
		log.Println("Skip Count:", retry.SkipCount)
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

	if *fDebug {
		printConfig(config)
	}

	return &config, nil
}
