// Copyright 2024 Google Inc. All Rights Reserved.
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

package cfg

import (
	"fmt"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/mitchellh/mapstructure"
)

func hookFunc() mapstructure.DecodeHookFuncType {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		s := data.(string)
		switch t {
		case reflect.TypeOf(Octal(0)):
			return strconv.ParseInt(s, 8, 32)
		case reflect.TypeOf(url.URL{}):
			u, err := url.Parse(s)
			if err != nil {
				return nil, err
			}
			return *u, nil
		case reflect.TypeOf(LogSeverity("")):
			level := strings.ToUpper(s)
			if !slices.Contains([]string{"TRACE", "DEBUG", "INFO", "WARNING", "ERROR", "OFF"}, level) {
				return nil, fmt.Errorf("invalid logseverity: %s", s)
			}
			return level, nil
		case reflect.TypeOf(Protocol("")):
			protocol := strings.ToLower(s)
			if !slices.Contains([]string{"http1", "http2", "grpc"}, protocol) {
				return nil, fmt.Errorf("invalid protocol: %s", s)
			}
			return protocol, nil
		case reflect.TypeOf(ResolvedPath("")):
			return util.GetResolvedPath(s)
		default:
			return data, nil
		}
	}
}

func DecodeHook() mapstructure.DecodeHookFunc {
	return mapstructure.ComposeDecodeHookFunc(
		mapstructure.TextUnmarshallerHookFunc(),
		hookFunc(),
		mapstructure.StringToTimeDurationHookFunc(), // default hook
		mapstructure.StringToSliceHookFunc(","),     // default hook
	)
}
