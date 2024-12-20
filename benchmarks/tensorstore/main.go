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
	"os"

	"github.com/bitfield/script"
	flag "github.com/spf13/pflag"
)

var (
	ip *int = flag.Int("flagname", 1234, "help message for flagname")
)

type multiReadConfig struct {
	fileIOConcurrency   int64
	maxInflightRequests int64
	numConfig           int64
}

func multiReadBenchmark(config *multiReadConfig) float64 {
	//script.Exec()
}

func setup() string {
	if p := script.Exec("apt install python3.10-dev g++ git"); p.ExitStatus() != 0 {
		panic("Error occurred while installing dependencies")
	}
	tempDir, err := os.MkdirTemp("", "tensorstore")
	if err != nil {
		panic(err)
	}
	if p := script.Exec(fmt.Sprintf("git clone https://github.com/google/tensorstore.git %s/", tempDir)); p.ExitStatus() != 0 {
		panic("Error while cloning the repository")
	}
	return tempDir
}

func main() {
	setup()

	//multiReadBenchmark()
}
