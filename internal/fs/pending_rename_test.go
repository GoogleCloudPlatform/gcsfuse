package fs

import (
	"context"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
	"github.com/stretchr/testify/assert"
)

func TestContextRenameTxID(t *testing.T) {
	ctx := context.Background()
	assert.Empty(t, renameTxIDFromContext(ctx))

	txID := "test-tx-id-123"
	ctx = contextWithRenameTxID(ctx, txID)
	assert.Equal(t, txID, renameTxIDFromContext(ctx))
}

func TestPendingRenameRegisterAndUnregister(t *testing.T) {
	fs := &fileSystem{
		pendingRenames: make(map[inode.Name]*renameWaitChannel),
	}
	fs.mu = locker.New("FS", nil)

	oldPath := inode.NewDirName(inode.NewRootName(""), "old_dir")
	newPath := inode.NewDirName(inode.NewRootName(""), "new_dir")

	txID := "test-tx"
	ch1, ch2 := fs.registerPendingRename(txID, oldPath, newPath)

	assert.NotNil(t, ch1)
	assert.NotNil(t, ch2)
	assert.Equal(t, txID, ch1.txID)
	assert.Equal(t, txID, ch2.txID)

	// Check that paths are registered in pendingRenames.
	firstPath, secondPath := oldPath, newPath
	if firstPath.LocalName() > secondPath.LocalName() {
		firstPath, secondPath = secondPath, firstPath
	}

	fs.mu.Lock()
	assert.Equal(t, ch1, fs.pendingRenames[firstPath])
	assert.Equal(t, ch2, fs.pendingRenames[secondPath])
	fs.mu.Unlock()

	// Verify that closing wait channels works upon unregistration.
	fs.unregisterPendingRename(oldPath, newPath, ch1, ch2)

	// Channels should be closed.
	_, open1 := <-ch1.done
	_, open2 := <-ch2.done
	assert.False(t, open1)
	assert.False(t, open2)

	// Check they are deleted from pendingRenames map.
	fs.mu.Lock()
	_, exists1 := fs.pendingRenames[firstPath]
	_, exists2 := fs.pendingRenames[secondPath]
	fs.mu.Unlock()
	assert.False(t, exists1)
	assert.False(t, exists2)
}
