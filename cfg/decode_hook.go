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
	"net/url"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

func stringToURLHookFunc() mapstructure.DecodeHookFuncType {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(&url.URL{}) {
			return data, nil
		}
		s := data.(string)
		u, err := url.Parse(s)
		if err != nil {
			return nil, err
		}
		return u, nil
	}
}

// DecodeHook will be called by Viper while constructing the config object.
func DecodeHook() mapstructure.DecodeHookFunc {
	return mapstructure.ComposeDecodeHookFunc(
		mapstructure.TextUnmarshallerHookFunc(),
		stringToURLHookFunc(),
		mapstructure.StringToTimeDurationHookFunc(), // default hook
		mapstructure.StringToSliceHookFunc(","),     // default hook
	)
}
