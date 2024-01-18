// Copyright 2023 Google Inc. All Rights Reserved.
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

package util

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
)

const GCSFUSE_PARENT_PROCESS_DIR = "gcsfuse-parent-process-dir"

// Constants for read types - Sequential/Random
const Sequential = "Sequential"
const Random = "Random"

// 1. Returns the same filepath in case of absolute path or empty filename.
// 2. For child process, it resolves relative path like, ./test.txt, test.txt
// ../test.txt etc, with respect to GCSFUSE_PARENT_PROCESS_DIR
// because we execute the child process from different directory and input
// files are provided with respect to GCSFUSE_PARENT_PROCESS_DIR.
// 3. For relative path starting with ~, it resolves with respect to home dir.
func GetResolvedPath(filePath string) (resolvedPath string, err error) {
	if filePath == "" || path.IsAbs(filePath) {
		resolvedPath = filePath
		return
	}

	// Relative path starting with tilda (~)
	if strings.HasPrefix(filePath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("fetch home dir: %w", err)
		}
		return filepath.Join(homeDir, filePath[2:]), err
	}

	// We reach here, when relative path starts with . or .. or other than (/ or ~)
	gcsfuseParentProcessDir, _ := os.LookupEnv(GCSFUSE_PARENT_PROCESS_DIR)
	gcsfuseParentProcessDir = strings.TrimSpace(gcsfuseParentProcessDir)
	if gcsfuseParentProcessDir == "" {
		return filepath.Abs(filePath)
	} else {
		return filepath.Join(gcsfuseParentProcessDir, filePath), err
	}
}

func ResolveFilePath(filePath string, configKey string) (resolvedPath string, err error) {
	resolvedPath, err = GetResolvedPath(filePath)
	if filePath == resolvedPath || err != nil {
		return
	}

	logger.Infof("Value of [%s] resolved from [%s] to [%s]\n", configKey, filePath, resolvedPath)
	return resolvedPath, nil
}

// ResolveConfigFilePaths resolved the config file paths specified in the config file.
func ResolveConfigFilePaths(config *config.MountConfig) (err error) {
	config.LogConfig.FilePath, err = ResolveFilePath(config.LogConfig.FilePath, "logging: file")
	if err != nil {
		return
	}
	return
}

// Stringify marshals an object (only exported attribute) to a JSON string. If marshalling fails, it returns an empty string.
func Stringify(input any) string {
	inputBytes, err := json.Marshal(input)

	if err != nil {
		logger.Warnf("Error in Stringify %v", err)
		return ""
	}
	return string(inputBytes)
}

// DeepSizeof returns the size of given struct/pointer/slice etc.
// including recursively adding the variable/struct/pointer/slice pointed/referred to
// by this variable.
// For pointers, includes the size of the pointer as well as the DeepSizeof of the
//
//	object pointed to by the pointer if it is not nil.
//
// For built-in types like integers, boolean etc. the output of this is same as that of
// unsafe.Sizeof(v).
// This does not exactly equal the actual size on memory, because it doesn't account for
// memory padding and alignments, but it is close to that for large structures.
// original source of logic: https://stackoverflow.com/questions/51431933/how-to-get-size-of-struct-containing-data-structures-in-go
func DeepSizeof(v any) int {
	size := int(reflect.TypeOf(v).Size())
	kind := reflect.TypeOf(v).Kind()

	// dereferencing begins
	if kind == reflect.Pointer {
		s := reflect.ValueOf(v)
		if !s.IsNil() {
			v = reflect.ValueOf(v).Elem().Interface()
			size += int(reflect.TypeOf(v).Size())
			kind = reflect.TypeOf(v).Kind()
		}
	}
	switch kind {
	case reflect.Int, reflect.Uint, reflect.Bool, reflect.Uint8, reflect.Int8, reflect.Uint16, reflect.Int16, reflect.Uint32, reflect.Int32, reflect.Uint64, reflect.Int64, reflect.Float32, reflect.Float64, reflect.Pointer:
		break
	case reflect.Slice:
		s := reflect.ValueOf(v)
		for i := 0; i < s.Len(); i++ {
			size += DeepSizeof(s.Index(i).Interface())
		}
	case reflect.Map:
		s := reflect.ValueOf(v)
		keys := s.MapKeys()
		size += int(float64(len(keys)) * 10.79) // approximation from https://golang.org/src/runtime/hashmap.go
		for i := range keys {
			size += DeepSizeof(keys[i].Interface()) + DeepSizeof(s.MapIndex(keys[i]).Interface())
		}
	case reflect.String:
		size += reflect.ValueOf(v).Len()
	case reflect.Struct:
		s := reflect.ValueOf(v)
		for i := 0; i < s.NumField(); i++ {
			if s.Field(i).CanInterface() {
				size -= int(s.Field(i).Type().Size()) // to reduce duplication, as it has been accounted already in reflect.TypeOf(v).Size()
				// at the top and will be accounted again in s.Field(i).Interface() .
				size += DeepSizeof(s.Field(i).Interface())
			}
		}
	case reflect.Interface:
		s := reflect.ValueOf(v)
		for i := 0; i < s.NumField(); i++ {
			size += DeepSizeof(s.Field(i).Interface())
		}
	default:
		panic(fmt.Sprintf("Unsupported type: %v", kind))
	}

	return size
}
