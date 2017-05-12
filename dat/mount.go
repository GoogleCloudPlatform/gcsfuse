package dat

import (
	"fmt"
	"os"

	"github.com/googlecloudplatform/gcsfuse/internal/fs"
	"github.com/googlecloudplatform/gcsfuse/internal/mount"
	"github.com/googlecloudplatform/gcsfuse/internal/perms"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fsutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/timeutil"
	"time"
)

// Mount the file system based on the supplied arguments, returning a
// fuse.MountedFileSystem that can be joined to wait for unmounting.
func CreateServer(
	bucket gcs.Bucket,
	flags *FlagStorage) (srv fuse.Server, err error) {
	// Sanity check: make sure the temporary directory exists and is writable
	// currently. This gives a better user experience than harder to debug EIO
	// errors when reading files in the future.
	if flags.TempDir != "" {
		var f *os.File
		f, err = fsutil.AnonymousFile(flags.TempDir)
		f.Close()

		if err != nil {
			err = fmt.Errorf(
				"Error writing to temporary directory (%q); are you sure it exists "+
					"with the correct permissions?",
				err.Error())
			return
		}
	}

	// Find the current process's UID and GID. If it was invoked as root and the
	// user hasn't explicitly overridden --uid, everything is going to be owned
	// by root. This is probably not what the user wants, so print a warning.
	uid, gid, err := perms.MyUserAndGroup()
	if err != nil {
		err = fmt.Errorf("MyUserAndGroup: %v", err)
		return
	}

	if uid == 0 && flags.Uid < 0 {
		fmt.Fprintln(os.Stderr, `
WARNING: gcsfuse invoked as root. This will cause all files to be owned by
root. If this is not what you intended, invoke gcsfuse as the user that will
be interacting with the file system.
`)
	}

	// Choose UID and GID.
	if flags.Uid >= 0 {
		uid = uint32(flags.Uid)
	}

	if flags.Gid >= 0 {
		gid = uint32(flags.Gid)
	}

	// Create a file system server.
	serverCfg := &fs.ServerConfig{
		CacheClock:             timeutil.RealClock(),
		Bucket:                 bucket,
		TempDir:                flags.TempDir,
		ImplicitDirectories:    flags.ImplicitDirs,
		InodeAttributeCacheTTL: flags.StatCacheTTL,
		DirTypeCacheTTL:        flags.TypeCacheTTL,
		Uid:                    uid,
		Gid:                    gid,
		FilePerms:              os.FileMode(flags.FileMode),
		DirPerms:               os.FileMode(flags.DirMode),

		AppendThreshold: 1 << 21, // 2 MiB, a total guess.
		TmpObjectPrefix: ".gcsfuse_tmp/",
	}

	server, err := fs.NewServer(serverCfg)
	if err != nil {
		return nil, fmt.Errorf("fs.NewServer: %v", err)
	}
	return server, nil

}

type FlagStorage struct {
	Foreground bool

	// File system
	MountOptions map[string]string
	MountOpts    string
	DirMode      os.FileMode
	FileMode     os.FileMode
	Uid          int64
	Gid          int64
	ImplicitDirs bool
	OnlyDir      string

	// GCS
	KeyFile                            string
	EgressBandwidthLimitBytesPerSecond float64
	OpRateLimitHz                      float64

	// Tuning
	StatCacheTTL time.Duration
	TypeCacheTTL time.Duration
	TempDir      string

	// Debugging
	DebugFuse       bool
	DebugGCS        bool
	DebugHTTP       bool
	DebugInvariants bool
}

func ParseOptions(m map[string]string, s string) {
	mount.ParseOptions(m, s)
}
