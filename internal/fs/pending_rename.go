package fs

import (
	"context"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
)

func (fs *fileSystem) registerPendingRename(
	txID string,
	oldPath, newPath inode.Name,
) (ch1, ch2 *renameWaitChannel) {
	firstPath, secondPath := oldPath, newPath
	if firstPath.LocalName() > secondPath.LocalName() {
		firstPath, secondPath = secondPath, firstPath
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	ch1 = &renameWaitChannel{txID: txID, done: make(chan struct{})}
	fs.pendingRenames[firstPath] = ch1
	ch2 = &renameWaitChannel{txID: txID, done: make(chan struct{})}
	fs.pendingRenames[secondPath] = ch2

	return ch1, ch2
}

func (fs *fileSystem) unregisterPendingRename(
	oldPath, newPath inode.Name,
	ch1, ch2 *renameWaitChannel,
) {
	firstPath, secondPath := oldPath, newPath
	if firstPath.LocalName() > secondPath.LocalName() {
		firstPath, secondPath = secondPath, firstPath
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	delete(fs.pendingRenames, firstPath)
	delete(fs.pendingRenames, secondPath)
	if ch1 != nil {
		close(ch1.done)
	}
	if ch2 != nil {
		close(ch2.done)
	}
}

func contextWithRenameTxID(ctx context.Context, txID string) context.Context {
	return context.WithValue(ctx, renameTxKey{}, txID)
}

func renameTxIDFromContext(ctx context.Context) string {
	if val := ctx.Value(renameTxKey{}); val != nil {
		return val.(string)
	}
	return ""
}
