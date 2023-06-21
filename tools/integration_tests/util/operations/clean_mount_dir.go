package clean_mount_dir

import (
	"log"
	"os"
	"path"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

// Clean mounted directory
func CleanMntDir() {
	dir, err := os.ReadDir(setup.MntDir())
	if err != nil {
		log.Printf("Error in reading directory: %v", err)
	}

	log.Print(len(dir))
	for _, d := range dir {
		err := os.RemoveAll(path.Join([]string{setup.MntDir(), d.Name()}...))
		if err != nil {
			log.Printf("Error in removing directory: %v", err)
		}
	}
}
