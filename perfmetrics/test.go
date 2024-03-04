package main

import (
	"log"
	"os/exec"
)

func RunScriptForTestData(args ...string) {
	cmd := exec.Command("/bin/bash", args...)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("Error: %s", out)
		panic(err)
	}
}

func main(){
	RunScriptForTestData("x.sh")
}