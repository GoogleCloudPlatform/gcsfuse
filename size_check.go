package main

import (
	"fmt"
	"time"
	"unsafe"
    "github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

type statCacheEntryType int

type entry struct {
	m          *gcs.MinObject
	f          *gcs.Folder
	expiration time.Time
	key        string
	entryType  statCacheEntryType
}

type oldEntry struct {
	m          *gcs.MinObject
	f          *gcs.Folder
	expiration time.Time
	key        string
}

func main() {
	var e entry
	var oe oldEntry
	fmt.Printf("Old entry size: %d\n", unsafe.Sizeof(oe))
	fmt.Printf("New entry size: %d\n", unsafe.Sizeof(e))
}
