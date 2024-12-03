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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewOperationManager(t *testing.T) {
	config := Config{
		RetryConfig: []RetryConfig{
			{Method: "JsonCreate", RetryInstruction: "return-503", RetryCount: 2, SkipCount: 1},
			{Method: "JsonStat", RetryInstruction: "stall-33s-after-20K", RetryCount: 3, SkipCount: 0},
		},
	}

	om := NewOperationManager(config)

	// Assert that retryConfigs are initialized correctly
	assert.Len(t, om.retryConfigs, 2)
	assert.Len(t, om.retryConfigs["JsonCreate"], 1)
	assert.Len(t, om.retryConfigs["JsonStat"], 1)

	assert.Equal(t, "return-503", om.retryConfigs["JsonCreate"][0].RetryInstruction)
	assert.Equal(t, "stall-33s-after-20K", om.retryConfigs["JsonStat"][0].RetryInstruction)
}

func TestRetrieveOperation(t *testing.T) {
	t.Run("One config test", func(t *testing.T) {
		config := Config{
			RetryConfig: []RetryConfig{
				{Method: "JsonCreate", RetryInstruction: "return-503", RetryCount: 2, SkipCount: 1},
			},
		}
		om := NewOperationManager(config)

		// First call: Skip count is decremented, so no retry instruction should be returned
		result := om.retrieveOperation("JsonCreate")
		assert.Equal(t, "", result, "Expected empty result due to SkipCount")

		// Second call: Retry instruction should be returned
		result = om.retrieveOperation("JsonCreate")
		assert.Equal(t, "return-503", result, "Expected 'return-503' as RetryInstruction")

		// Third call: Retry instruction should be returned again
		result = om.retrieveOperation("JsonCreate")
		assert.Equal(t, "return-503", result, "Expected 'return-503' as RetryInstruction")

		// Fourth call: Retry count is exhausted, so no retry instruction should be returned
		result = om.retrieveOperation("JsonCreate")
		assert.Equal(t, "", result, "Expected empty result as RetryCount is exhausted")
	})

	t.Run("Multiple config tests with same request types", func(t *testing.T) {
		// Initialize OperationManager with two retry configs
		config := Config{
			RetryConfig: []RetryConfig{
				{Method: "RequestTypeA", RetryInstruction: "retry-503", RetryCount: 2, SkipCount: 1},
				{Method: "RequestTypeA", RetryInstruction: "retry-202", RetryCount: 1, SkipCount: 0},
			},
		}
		om := NewOperationManager(config)

		// Test for RequestTypeA
		// First call: SkipCount is decremented, so no retry instruction should be returned
		result := om.retrieveOperation("RequestTypeA")
		assert.Equal(t, "", result, "Expected no result due to SkipCount")

		// Second call: First retry instruction should be returned
		result = om.retrieveOperation("RequestTypeA")
		assert.Equal(t, "retry-503", result, "Expected 'retry-503' as RetryInstruction")

		// Third call: Second retry instruction should be returned
		result = om.retrieveOperation("RequestTypeA")
		assert.Equal(t, "retry-503", result, "Expected 'retry-503' as RetryInstruction")

		// Fourth call: Move to the second config for RequestTypeA
		result = om.retrieveOperation("RequestTypeA")
		assert.Equal(t, "retry-202", result, "Expected 'retry-202' as RetryInstruction")

		// Fifth call: All retry instructions exhausted, so no result
		result = om.retrieveOperation("RequestTypeA")
		assert.Equal(t, "", result, "Expected no result as all retries are exhausted")
	})

	t.Run("Multiple config tests with different request types", func(t *testing.T) {
		// Initialize OperationManager with two retry configs
		config := Config{
			RetryConfig: []RetryConfig{
				{Method: "RequestTypeA", RetryInstruction: "retry-503", RetryCount: 2, SkipCount: 1},
				{Method: "RequestTypeB", RetryInstruction: "retry-202", RetryCount: 1, SkipCount: 0},
			},
		}
		om := NewOperationManager(config)

		// Test for RequestTypeA
		// First call: SkipCount is decremented, so no retry instruction should be returned
		result := om.retrieveOperation("RequestTypeA")
		assert.Equal(t, "", result, "Expected no result due to SkipCount")

		// Second call: First retry instruction should be returned
		result = om.retrieveOperation("RequestTypeA")
		assert.Equal(t, "retry-503", result, "Expected 'retry-503' as RetryInstruction")

		// Third call: Second retry instruction should be returned
		result = om.retrieveOperation("RequestTypeA")
		assert.Equal(t, "retry-503", result, "Expected 'retry-503' as RetryInstruction")

		// Fourth call: Move to the second config for RequestTypeA
		result = om.retrieveOperation("RequestTypeB")
		assert.Equal(t, "retry-202", result, "Expected 'retry-202' as RetryInstruction")
	})
}

func TestAddRetryConfig(t *testing.T) {
	om := &OperationManager{
		retryConfigs: make(map[RequestType][]RetryConfig),
	}

	retryConfig := RetryConfig{Method: "JsonUpdate", RetryInstruction: "retry-202", RetryCount: 1, SkipCount: 0}
	om.addRetryConfig(retryConfig)

	// Assert the retryConfig is added to the map
	assert.Len(t, om.retryConfigs, 1)
	assert.Len(t, om.retryConfigs["JsonUpdate"], 1)
	assert.Equal(t, "retry-202", om.retryConfigs["JsonUpdate"][0].RetryInstruction)

	// Add another retryConfig for the same method
	retryConfig2 := RetryConfig{Method: "JsonUpdate", RetryInstruction: "retry-503", RetryCount: 2, SkipCount: 1}
	om.addRetryConfig(retryConfig2)

	// Assert both retryConfigs are stored under the same key
	assert.Len(t, om.retryConfigs["JsonUpdate"], 2)
	assert.Equal(t, "retry-202", om.retryConfigs["JsonUpdate"][0].RetryInstruction)
	assert.Equal(t, "retry-503", om.retryConfigs["JsonUpdate"][1].RetryInstruction)
}
