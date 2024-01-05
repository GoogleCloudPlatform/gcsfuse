package main

import (
	"encoding/json"
	"fmt"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/log_parser"
)

func main() {
	mapstruct, err := log_parser.ParseLogFile("/usr/local/google/home/ashmeen/Documents/log.json")
	if err != nil {
		fmt.Println(err)
	}

	jsonObject, _ := json.MarshalIndent(mapstruct, "", "  ")
	fmt.Println(string(jsonObject))
}
