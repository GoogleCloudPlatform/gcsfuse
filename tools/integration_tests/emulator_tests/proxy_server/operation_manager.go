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
	"log"
	"sync"
)

type OperationManager struct {
	retryConfigs map[RequestType][]RetryConfig
	mu           sync.Mutex
}

func NewOperationManager(config Config) *OperationManager {
	rc := make(map[RequestType][]RetryConfig)
	om := &OperationManager{
		retryConfigs: rc,
	}
	for _, retryConfig := range config.RetryConfig {
		om.addRetryConfig(retryConfig)
	}

	if *fDebug {
		println(om)
	}

	return om
}

// Empty string represent there is no plantation required.
func (om *OperationManager) retrieveOperation(requestType RequestType) string {
	om.mu.Lock()
	defer om.mu.Unlock()

	configs, ok := om.retryConfigs[requestType]
	if !ok {
		return ""
	}

	for len(configs) > 0 {
		cc := &configs[0]
		log.Println("retryCount: ", cc.RetryCount)
		if cc.SkipCount > 0 {
			cc.SkipCount--
			return ""
		} else if cc.RetryCount > 0 {
			cc.RetryCount--
			return cc.RetryInstruction
		} else {
			configs = configs[1:]
			om.retryConfigs[requestType] = configs
		}
	}
	return ""
}

func (om *OperationManager) addRetryConfig(rc RetryConfig) {
	rt := RequestType(rc.Method)
	if *fDebug {
		println(rt)
	}
	if om.retryConfigs[rt] != nil {
		// Key exists, append the new retryConfig to the existing list
		om.retryConfigs[rt] = append(om.retryConfigs[rt], rc)
	} else {
		// Key doesn't exist, getRetryID a new list with the retryConfig
		om.retryConfigs[rt] = []RetryConfig{rc}
	}
}
