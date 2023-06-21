package clean_mount_dir

import (
	"log"
	"os"
	"path"
)

// Clean mounted directory
func CleanMntDir(mntDir string) {
	dir, err := os.ReadDir(mntDir)
	if err != nil {
		log.Printf("Error in reading directory: %v", err)
	}

	log.Print(len(dir))
	for _, d := range dir {
		err := os.RemoveAll(path.Join([]string{mntDir, d.Name()}...))
		if err != nil {
			log.Printf("Error in removing directory: %v", err)
		}
	}
}
