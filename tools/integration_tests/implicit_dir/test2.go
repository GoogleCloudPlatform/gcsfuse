package main

import (
	"log"
	"os"
	"path"
)

func main() {

	os.Mkdir("/usr/local/google/home/tulsishah/gcs/implicitDirectory", 777)
	os.Create("/usr/local/google/home/tulsishah/gcs/implicitDirectory/f1.txt")
	os.Mkdir("/usr/local/google/home/tulsishah/gcs/implicitDirectory/implicitSubDirectory", 777)

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

	os.Mkdir("/usr/local/google/home/tulsishah/gcs/implicitDirectory", 777)
	os.Create("/usr/local/google/home/tulsishah/gcs/implicitDirectory/f1.txt")
	os.Mkdir("/usr/local/google/home/tulsishah/gcs/implicitDirectory/implicitSubDirectory", 777)

	_, err = os.Stat("/usr/local/google/home/tulsishah/gcs/implicitDirectory/implicitSubDirectory/")
	if err != nil {
		log.Printf("Stating file at : %v", err)
	}
}
