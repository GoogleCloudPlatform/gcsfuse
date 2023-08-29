package main

import (
	"bytes"
	"fmt"
	"os/exec"
)

func main() {
	cmd := exec.Command("/bin/bash", "-c", "gcloud iam service-accounts keys create ~/creds.json --iam-account=creds-test-gcsfuse-jilyizenaw@gcs-fuse-test-ml.iam.gserviceaccount.com")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		fmt.Println(stdout.Bytes())
		fmt.Println(stderr.Bytes())
	}
}
