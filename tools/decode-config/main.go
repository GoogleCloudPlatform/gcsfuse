// Copyright 2026 Google LLC
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
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run tools/decode-config/main.go <base64_config_string>")
		os.Exit(1)
	}

	input := strings.TrimSpace(os.Args[1])

	// Extract payload if embedded in a User-Agent log string.
	// Matches either (CfgJson:<base64>) or CfgJson:<base64>
	for _, marker := range []string{"(CfgJson:", "CfgJson:"} {
		if idx := strings.Index(input, marker); idx != -1 {
			input = input[idx+len(marker):]
			if endIdx := strings.IndexAny(input, " )"); endIdx != -1 {
				input = input[:endIdx]
			}
			break
		}
	}

	// Clean potential whitespace
	input = strings.TrimSpace(input)

	var decodedBytes []byte
	var decodeErr error
	decoders := []*base64.Encoding{
		base64.RawURLEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.StdEncoding,
	}
	for _, dec := range decoders {
		decodedBytes, decodeErr = dec.DecodeString(input)
		if decodeErr == nil {
			break
		}
	}
	if decodeErr != nil {
		fmt.Printf("Error: Failed to decode base64 input string: %v\n", decodeErr)
		os.Exit(1)
	}

	// Attempt 1: Try to decompress with Gzip (Gzipped JSON format)
	gzipReader, err := gzip.NewReader(bytes.NewReader(decodedBytes))
	if err != nil {
		fmt.Printf("[Debug] Gzip new reader error: %v\n", err)
	} else {
		var decompressedBuf bytes.Buffer
		_, err = decompressedBuf.ReadFrom(gzipReader)
		gzipReader.Close()
		if err != nil {
			fmt.Printf("[Debug] Gzip read error: %v\n", err)
		} else {
			var jsonMap map[string]any
			if err := json.Unmarshal(decompressedBuf.Bytes(), &jsonMap); err == nil {
				prettyJSON, err := json.MarshalIndent(jsonMap, "", "  ")
				if err != nil {
					fmt.Printf("Error pretty-printing decompressed JSON config: %v\n", err)
					os.Exit(1)
				}
				fmt.Println("Successfully decoded Base64-encoded Gzipped JSON config:")
				fmt.Println(string(prettyJSON))
				return
			} else {
				fmt.Printf("[Debug] JSON unmarshal error on decompressed bytes: %v\n", err)
			}
		}
	}

	// Attempt 2: Treat directly as raw JSON (Raw JSON format)
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

	fmt.Println("Error: Decoded bytes are neither valid Gzipped JSON nor valid Raw JSON.")
	os.Exit(1)
}
