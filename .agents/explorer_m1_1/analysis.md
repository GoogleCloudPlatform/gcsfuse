# Inode Read-Locking Optimization Analysis

This report analyzes the inode lookup and listing paths in `gcsfuse` to design and implement read-locking optimizations for concurrent lookups.

## 1. Symbol Locations

*   **`LookUpInode`**: `internal/fs/fs.go` (line 1912)
*   **`coreToDirentPlus`**: `internal/fs/fs.go` (line 1700)
*   **`lookUpOrCreateInodeIfNotStale`**: `internal/fs/fs.go` (line 1089)

---

## 2. Inode Lock Architectures & `locker.RWLocker`

*   **`locker.RWLocker`**: Defined in `internal/locker/rw_locker.go` as:
    ```go
    type RWLocker interface {
        sync.Locker
        RLock()
        RUnlock()
    }
    ```
*   **Directory Inodes**: Both `dirInode` (`internal/fs/inode/dir.go`) and `baseDirInode` (`internal/fs/inode/base_dir.go`) implement `RLock()` and `RUnlock()`, delegating to an internal `mu locker.RWLocker` field. Hence, they implement `locker.RWLocker` and support read-locking.
*   **File and Symlink Inodes**: `FileInode` (`internal/fs/inode/file.go`) uses an exclusive-only `syncutil.InvariantMutex`, and `SymlinkInode` (`internal/fs/inode/symlink.go`) uses `sync.Mutex`. Neither of these types implements `locker.RWLocker` or supports read-locking.

---

## 3. Proposal for Read-Locking and Lock Upgrades

### A. Atomic Lookup Count
Since multiple lookup or directory listing operations can return a directory inode in a read-locked state, incrementing/decrementing the lookup count concurrently on the same inode would result in a data race. To avoid this, we must make `lookupCount` in `internal/fs/inode/lookup_count.go` thread-safe using atomic operations.

**Proposed Changes to `internal/fs/inode/lookup_count.go`**:
```go
type lookupCount struct {
	id        fuseops.InodeID
	count     uint64
	destroyed uint32 // 0 for false, 1 for true
}

func (lc *lookupCount) Init(id fuseops.InodeID) {
	lc.id = id
}

func (lc *lookupCount) Inc() {
	if atomic.LoadUint32(&lc.destroyed) == 1 {
		panic(fmt.Sprintf("Inode %v has already been destroyed", lc.id))
	}
	atomic.AddUint64(&lc.count, 1)
}

func (lc *lookupCount) Dec(n uint64) (destroy bool) {
	if atomic.LoadUint32(&lc.destroyed) == 1 {
		panic(fmt.Sprintf("Inode %v has already been destroyed", lc.id))
	}
	for {
		val := atomic.LoadUint64(&lc.count)
		if n > val {
			panic(fmt.Sprintf("n is greater than lookup count: %v vs. %v", n, val))
		}
		newVal := val - n
		if atomic.CompareAndSwapUint64(&lc.count, val, newVal) {
			if newVal == 0 {
				atomic.StoreUint32(&lc.destroyed, 1)
				destroy = true
			}
			break
		}
	}
	return
}
```

### B. Read-Lock Option in `lookUpOrCreateInodeIfNotStale`
We propose adding a `readLock bool` parameter to `lookUpOrCreateInodeIfNotStale` and helper functions:
*   If `readLock` is true and the inode implements `locker.RWLocker`, we acquire a read lock.
*   Otherwise, we acquire a write/exclusive lock.

Helper lock/unlock abstractions:
```go
lock := func(i inode.Inode) bool {
    if readLock {
        if rw, ok := i.(locker.RWLocker); ok {
            rw.RLock()
            return false // read-locked
        }
    }
    i.Lock()
    return true // write-locked
}
```

### C. Lock Upgrading on Remote Size Changes
If a lookup finds that the child size has changed remotely (`cmp == 2` in `lookUpOrCreateInodeIfNotStale`):
1. If the inode was read-locked, we release the read lock and the filesystem lock (`fs.mu.Unlock()`).
2. We acquire the write lock (`existingInode.Lock()`) and re-acquire `fs.mu.Lock()`.
3. We verify if the index still points to this inode. If not, we release the write lock and retry the lookup loop.
4. We recheck if `cmp == 2`. If so, we call `existingInode.UpdateSize(oGen.Size)`.
5. If `readLock` was originally requested, we downgrade the lock back to a read lock (by releasing the write lock and acquiring the read lock) before returning, keeping the return state consistent.

### D. Function Signatures & Call-site Updates
*   **`lookUpOrCreateInodeIfNotStale(parInodeCtx context.Context, ic inode.Core, readLock bool)`**
*   **`createDirInode(ic inode.Core, inodes map[inode.Name]inode.DirInode, parInodeCtx context.Context, readLock bool)`**
*   **`lookUpOrCreateChildInode(ctx context.Context, parent inode.DirInode, childName string, readLock bool)`**
*   **`unlockAndMaybeDisposeOfInode(in inode.Inode, readLocked bool, err *error)`**
*   **`unlockAndDecrementLookupCount(in inode.Inode, N uint64, readLocked bool)`**

#### Paths that should request read-lock:
*   `LookUpInode` (R2)
*   `coreToDirentPlus` (R2)

#### Paths that must request write-lock:
*   `MkDir`
*   `CreateFile`
*   `Symlink`
*   `RmDir`
*   `Rename`
*   `ForgetInode`

---

## 4. Code Change Specifications for `internal/fs/fs.go`

Below are the detailed proposed implementations for modified functions in `internal/fs/fs.go`.

### `createDirInode`
```go
func (fs *fileSystem) createDirInode(ic inode.Core, inodes map[inode.Name]inode.DirInode, parInodeCtx context.Context, readLock bool) (inode.Inode, error) {
	if !ic.FullName.IsDir() {
		panic(fmt.Sprintf("Unexpected name for a directory: %q", ic.FullName))
	}

	var maxTriesToCreateInode = 3

	for range maxTriesToCreateInode {
		in, ok := (inodes)[ic.FullName]
		// Create a new inode when a folder is created first time, or when a folder is deleted and then recreated with the same name.
		if !ok || in.IsUnlinked() {
			in, err := fs.mintInode(ic, parInodeCtx)
			if err != nil {
				return nil, err
			}
			(inodes)[in.Name()] = in.(inode.DirInode)
			
			if readLock {
				if rw, ok := in.(locker.RWLocker); ok {
					rw.RLock()
				} else {
					in.Lock()
				}
			} else {
				in.Lock()
			}
			return in, nil
		}

		fs.mu.Unlock()
		if readLock {
			if rw, ok := in.(locker.RWLocker); ok {
				rw.RLock()
			} else {
				in.Lock()
			}
		} else {
			in.Lock()
		}
		fs.mu.Lock()

		if (inodes)[ic.FullName] != in {
			if readLock {
				if rw, ok := in.(locker.RWLocker); ok {
					rw.RUnlock()
				} else {
					in.Unlock()
				}
			} else {
				in.Unlock()
			}
			continue
		}

		return in, nil
	}

	return nil, fmt.Errorf("createDirInode: failed to create inode after %d tries", maxTriesToCreateInode)
}
```

### `lookUpOrCreateInodeIfNotStale`
```go
func (fs *fileSystem) lookUpOrCreateInodeIfNotStale(parInodeCtx context.Context, ic inode.Core, readLock bool) (in inode.Inode, err error) {

	if err := ic.SanityCheck(); err != nil {
		panic(err.Error())
	}

	// Ensure that no matter which inode we return, we increase its lookup count
	// on the way out and then release the file system lock.
	defer func() {
		if in != nil {
			in.IncrementLookupCount()
		}

		fs.mu.Unlock()
	}()

	fs.mu.Lock()

	// Handle Folders in hierarchical bucket.
	if ic.Folder != nil {
		return fs.createDirInode(ic, fs.folderInodes, parInodeCtx, readLock)
	}

	// Handle implicit directories.
	if ic.MinObject == nil {
		return fs.createDirInode(ic, fs.implicitDirInodes, parInodeCtx, readLock)
	}

	oGen := inode.Generation{
		Object:   ic.MinObject.Generation,
		Metadata: ic.MinObject.MetaGeneration,
		Size:     ic.MinObject.Size,
	}

	// Retry loop for the stale index entry case below. On entry, we hold fs.mu
	// but no inode lock.
	for {
		// Look at the current index entry.
		existingInode, ok := fs.generationBackedInodes[ic.FullName]

		// If we have no existing record, mint an inode and return it.
		if !ok {
			in, err = fs.mintInode(ic, parInodeCtx)
			if err != nil {
				return nil, err
			}
			fs.generationBackedInodes[in.Name()] = in.(inode.GenerationBackedInode)

			if readLock {
				if rw, ok := in.(locker.RWLocker); ok {
					rw.RLock()
				} else {
					in.Lock()
				}
			} else {
				in.Lock()
			}
			return in, nil
		}

		// Otherwise we need to read the inode's source generation below, which
		// requires the inode's lock. We must not hold the inode lock while
		// acquiring the file system lock, so drop it while acquiring the inode's
		// lock, then reacquire.
		fs.mu.Unlock()
		
		var heldWriteLock bool
		if readLock {
			if rw, ok := existingInode.(locker.RWLocker); ok {
				rw.RLock()
			} else {
				existingInode.Lock()
				heldWriteLock = true
			}
		} else {
			existingInode.Lock()
			heldWriteLock = true
		}
		
		fs.mu.Lock()

		// Check that the index still points at this inode. If not, it's possible
		// that the inode is in the process of being destroyed and is unsafe to
		// use. Go around and try again.
		if fs.generationBackedInodes[ic.FullName] != existingInode {
			if heldWriteLock {
				existingInode.Unlock()
			} else {
				existingInode.(locker.RWLocker).RUnlock()
			}
			continue
		}

		// Have we found the correct inode?
		cmp := oGen.Compare(existingInode.SourceGeneration())
		if cmp == 0 {
			in = existingInode
			return
		}

		// The existing inode has the same generation but a different size.
		// Update the size and return the existing inode.
		if cmp == 2 {
			if !heldWriteLock {
				// Safely upgrade the lock to a write lock
				existingInode.(locker.RWLocker).RUnlock()
				fs.mu.Unlock()
				
				existingInode.Lock()
				fs.mu.Lock()
				
				// Re-verify the inode under write lock
				if fs.generationBackedInodes[ic.FullName] != existingInode {
					existingInode.Unlock()
					continue
				}
				cmp = oGen.Compare(existingInode.SourceGeneration())
				if cmp != 2 {
					existingInode.Unlock()
					continue
				}
				heldWriteLock = true
			}
			
			logger.Warnf("The size of object has changed remotely at the same generation. Updating the existing inode to reflect the size change.\n")
			existingInode.UpdateSize(oGen.Size)
			
			// Downgrade back to read lock if originally requested
			if readLock {
				existingInode.Unlock()
				existingInode.(locker.RWLocker).RLock()
				heldWriteLock = false
			}
			
			in = existingInode
			return
		}

		// The existing inode is newer than the backing object. The caller
		// should call again with a newer backing object.
		if cmp == -1 {
			if heldWriteLock {
				existingInode.Unlock()
			} else {
				existingInode.(locker.RWLocker).RUnlock()
			}
			return
		}

		// The backing object is newer than the existing inode, while
		// holding the inode lock, excluding concurrent actions by the inode (in
		// particular concurrent calls to Sync, which changes generation numbers).
		// This means we've proven that the record cannot have been caused by the
		// inode's actions, and therefore this is not the inode we want.
		//
		// Replace it with a newly-mintend inode and then go around, acquiring its
		// lock in accordance with our lock ordering rules.
		if heldWriteLock {
			existingInode.Unlock()
		} else {
			existingInode.(locker.RWLocker).RUnlock()
		}

		in, err = fs.mintInode(ic, parInodeCtx)
		if err != nil {
			return nil, err
		}
		fs.generationBackedInodes[in.Name()] = in.(inode.GenerationBackedInode)

		continue
	}
}
```

### `unlockAndDecrementLookupCount`
```go
func (fs *fileSystem) unlockAndDecrementLookupCount(in inode.Inode, N uint64, readLocked bool) {
	name := in.Name()

	// Decrement the lookup count.
	shouldDestroy := in.DecrementLookupCount(N)

	// Update file system state, orphaning the inode if we're going to destroy it
	// below.
	if shouldDestroy {
		fs.mu.Lock()
		delete(fs.inodes, in.ID())

		// Update indexes if necessary.
		if fs.generationBackedInodes[name] == in {
			delete(fs.generationBackedInodes, name)
		}
		if fs.implicitDirInodes[name] == in {
			delete(fs.implicitDirInodes, name)
		}
		if fs.localFileInodes[name] == in {
			delete(fs.localFileInodes, name)
		}
		if fs.folderInodes[name] == in {
			delete(fs.folderInodes, name)
		}
		fs.mu.Unlock()
	}

	// Now we can destroy the inode if necessary.
	if shouldDestroy {
		destroyErr := in.Destroy()
		if destroyErr != nil {
			logger.Infof("Error destroying inode %q: %v", name, destroyErr)
		}
	}

	if readLocked {
		if rw, ok := in.(locker.RWLocker); ok {
			rw.RUnlock()
			return
		}
	}
	in.Unlock()
}
```

### `unlockAndMaybeDisposeOfInode`
```go
func (fs *fileSystem) unlockAndMaybeDisposeOfInode(
	in inode.Inode,
	readLocked bool,
	err *error) {
	// If there is no error, just unlock.
	if *err == nil {
		if readLocked {
			if rw, ok := in.(locker.RWLocker); ok {
				rw.RUnlock()
				return
			}
		}
		in.Unlock()
		return
	}

	// Otherwise, go through the decrement helper
	fs.unlockAndDecrementLookupCount(in, 1, readLocked)
}
```
