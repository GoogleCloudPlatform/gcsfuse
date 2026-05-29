/*
 * Copyright 2026 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg/pb"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run tools/decode-config/main.go <base64_config_string>")
		os.Exit(1)
	}

	input := strings.TrimSpace(os.Args[1])

	// If the input is wrapped inside a full user agent log string, extract the CfgProto portion.
	// E.g., "... (CfgProto:eyJBcHBOYW1l...) ..."
	if idx := strings.Index(input, "(CfgProto:"); idx != -1 {
		input = input[idx+len("(CfgProto:"):]
		if endIdx := strings.Index(input, ")"); endIdx != -1 {
			input = input[:endIdx]
		}
	} else if idx := strings.Index(input, "CfgProto:"); idx != -1 {
		input = input[idx+len("CfgProto:"):]
		if endIdx := strings.Index(input, " "); endIdx != -1 {
			input = input[:endIdx]
		}
	}

	// Clean potential whitespace or trailing characters
	input = strings.TrimSpace(input)

	decodedBytes, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		// Try URL-safe encoding just in case
		decodedBytes, err = base64.URLEncoding.DecodeString(input)
		if err != nil {
			fmt.Printf("Error: Failed to decode base64 input string: %v\n", err)
			os.Exit(1)
		}
	}

	// Attempt 1: Check if the decoded bytes are valid JSON (older base64 format)
	var jsonMap map[string]any
	if err := json.Unmarshal(decodedBytes, &jsonMap); err == nil {
		prettyJSON, err := json.MarshalIndent(jsonMap, "", "  ")
		if err != nil {
			fmt.Printf("Error pretty-printing JSON config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Successfully decoded Base64-encoded JSON config:")
		fmt.Println(string(prettyJSON))
		return
	}

	// Attempt 2: Treat as marshaled Protobuf binary (new Base64-proto format)
	configProto := &pb.Config{}
	if err := proto.Unmarshal(decodedBytes, configProto); err != nil {
		fmt.Printf("Error: Decoded bytes are neither valid JSON nor a valid serialized Config Protobuf: %v\n", err)
		os.Exit(1)
	}

	// Convert Protobuf to pretty-printed JSON using protojson
	marshaler := protojson.MarshalOptions{
		Multiline:       true,
		Indent:          "  ",
		EmitUnpopulated: false,
	}
	prettyJSONBytes, err := marshaler.Marshal(configProto)
	if err != nil {
		fmt.Printf("Error converting Protobuf config to JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully decoded Base64-encoded Protobuf config:")
	fmt.Println(string(prettyJSONBytes))
}
