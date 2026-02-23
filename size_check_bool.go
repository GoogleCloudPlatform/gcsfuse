package main

import (
	"fmt"
	"time"
	"unsafe"
    "github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

type entry struct {
	m          *gcs.MinObject
	f          *gcs.Folder
	expiration time.Time
	key        string
	isImplicitDir bool
}

func main() {
	var e entry
	fmt.Printf("Bool entry size: %d\n", unsafe.Sizeof(e))
}
