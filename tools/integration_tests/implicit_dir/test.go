package main

import (
	"log"
	"os"
	"path"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func main() {
	setup.RunScriptForTestData("delete_objects.sh", "tulsishah-test")

	setup.RunScriptForTestData("create_objects.sh", "tulsishah-test")

	err := os.Mkdir("/usr/local/google/home/tulsishah/gcs/implicitDirectory/A", 777)
	if err != nil {
		log.Printf("Error in creating directory: %v", err)
	}

	filePath := path.Join("/usr/local/google/home/tulsishah/gcs/implicitDirectory/A/a.txt")
	_, err = os.Create(filePath)
	if err != nil {
		log.Printf("Create file at : %v", err)
	}

	os.RemoveAll("/usr/local/google/home/tulsishah/gcs/implicitDirectory")

	setup.RunScriptForTestData("delete_objects.sh", "tulsishah-test")

	setup.RunScriptForTestData("create_objects.sh", "tulsishah-test")

	err = os.Mkdir("/usr/local/google/home/tulsishah/gcs/A", 777)
	if err != nil {
		log.Printf("Error in creating directory: %v", err)
	}

	filePath = path.Join("/usr/local/google/home/tulsishah/gcs/A/a.txt")
	_, err = os.Create(filePath)
	if err != nil {
		log.Printf("Create file at : %v", err)
	}

	_, err = os.Stat("/usr/local/google/home/tulsishah/gcs/implicitDirectory/implicitSubDirectory/")
	if err != nil {
		log.Printf("Stating file at : %v", err)
	}
}
