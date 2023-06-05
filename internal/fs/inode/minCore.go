package inode

import (
	"fmt"

	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/jacobsa/gcloud/gcs"

)




// IsSymlink Does the supplied object represent a symlink inode?
func MinObjectIsSymlink(o *gcs.MinObject) bool {
	_, ok := o.Metadata[SymlinkMetadataKey]
	return ok
}


// Core contains critical information about an inode before its creation.
type MinCore struct {
	// The full name of the file or directory. Required for all inodes.
	FullName Name

	// The bucket that backs up the inode. Required for all inodes except the
	// base directory that holds all the buckets mounted.
	Bucket *gcsx.SyncerBucket

	// The GCS object in the bucket above that backs up the inode. Can be empty
	// if the inode is the base directory or an implicit directory.
	Object *gcs.MinObject
}






// Exists returns true iff the back object exists implicitly or explicitly.
func (c *MinCore) MinObjectExists() bool {
	return c != nil
}

func (c *MinCore) MinObjectType() Type {
	switch {
	case c == nil:
		return UnknownType
	case c.Object == nil:
		return ImplicitDirType
	case c.FullName.IsDir():
		return ExplicitDirType
	case MinObjectIsSymlink(c.Object):
		return SymlinkType
	default:
		return RegularFileType
	}
}

// SanityCheck returns an error if the object is conflicting with itself, which
// means the metadata of the file system is broken.
func (c MinCore) SanityCheckMinCore() error {
	if c.Object != nil && c.FullName.objectName != c.Object.Name {
		return fmt.Errorf("inode name %q mismatches object name %q", c.FullName, c.Object.Name)
	}
	if c.Object == nil && !c.FullName.IsDir() {
		return fmt.Errorf("object missing for %q", c.FullName)
	}
	return nil
}