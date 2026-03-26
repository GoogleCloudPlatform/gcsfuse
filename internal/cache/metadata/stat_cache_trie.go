package metadata

import (
	"container/list"
	"sync"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

// statCacheTrie is an implementation of StatCache based on a Radix Trie,
// supporting LRU eviction.
type statCacheTrie struct {
	mu sync.RWMutex

	// The maximum size of the cache in bytes.
	maxSize uint64

	// The current total size of elements in the cache.
	currentSize uint64

	// The root of the radix trie
	root *trieNode

	// List of cache entries (type *trieNode), with least recently used at the tail.
	entries list.List
}

type trieChild struct {
	b byte
	n *trieNode
}

type trieNode struct {
	// The prefix of the string associated with this node
	prefix string

	// The character mapped to the next node in the path
	children []trieChild

	// The cached entry (if any)
	hasVal bool
	val    entry

	// Pointer to the list element in the LRU tracking list
	lruElem *list.Element

	// Parent pointer to allow bottom-up pruning
	parent *trieNode

	// The character this node is registered under in the parent's children map
	charInParent byte
}

func newTrieNode(prefix string) *trieNode {
	return &trieNode{
		prefix: prefix,
	}
}

// NewStatCacheTrie creates a new stat cache backed by a radix trie.
func NewStatCacheTrie(maxSize uint64) StatCache {
	return &statCacheTrie{
		maxSize: maxSize,
		root:    newTrieNode(""),
	}
}

// longestCommonPrefix length
func longestCommonPrefix(k1, k2 string) int {
	maxLen := len(k1)
	if len(k2) < maxLen {
		maxLen = len(k2)
	}
	for i := 0; i < maxLen; i++ {
		if k1[i] != k2[i] {
			return i
		}
	}
	return maxLen
}

// findNode returns the node and its parent matching exactly the key,
// or the closest node and the remaining key to be inserted.
func (sc *statCacheTrie) findNode(key string) (node *trieNode, remainingKey string, exactMatch bool) {
	node = sc.root
	remainingKey = key

	for len(remainingKey) > 0 {
		c := remainingKey[0]
		var child *trieNode
		for i := range node.children {
			if node.children[i].b == c {
				child = node.children[i].n
				break
			}
		}
		if child == nil {
			return node, remainingKey, false
		}

		lcp := longestCommonPrefix(child.prefix, remainingKey)

		if lcp < len(child.prefix) {
			// partial match, we need to split
			return child, remainingKey, false
		}

		// Full match of the child's prefix
		remainingKey = remainingKey[lcp:]
		node = child
	}

	return node, "", true
}

func (sc *statCacheTrie) insertNode(key string, val entry) {
	node, remainingKey, exactMatch := sc.findNode(key)

	var targetNode *trieNode

	if exactMatch {
		// Replace or set value
		targetNode = node
		if targetNode.hasVal {
			sc.currentSize -= targetNode.val.Size()
		}
		targetNode.val = val
		targetNode.hasVal = true
	} else if node == sc.root || len(remainingKey) > 0 && longestCommonPrefix(node.prefix, remainingKey) == 0 {
		// We are at a node (like root or another fully matched node) and the next char isn't in children.
		// node is the parent for the new node.
		newNode := newTrieNode(remainingKey)
		newNode.val = val
		newNode.hasVal = true
		newNode.parent = node
		newNode.charInParent = remainingKey[0]
		node.children = append(node.children, trieChild{remainingKey[0], newNode})
		targetNode = newNode
	} else {
		// We have a partial match with an existing child node
		lcp := longestCommonPrefix(node.prefix, remainingKey)

		if lcp == len(node.prefix) {
			// Actually matched the entire prefix of the current node (can happen if node returned by findNode was actually fully matched but next char wasn't in its children map)
			if lcp < len(remainingKey) {
				newNode := newTrieNode(remainingKey[lcp:])
				newNode.val = val
				newNode.hasVal = true
				newNode.parent = node
				newNode.charInParent = remainingKey[lcp]
				node.children = append(node.children, trieChild{remainingKey[lcp], newNode})
				targetNode = newNode
			} else {
				node.val = val
				node.hasVal = true
				targetNode = node
			}
		} else {
			// Create a split node
			splitNode := newTrieNode(node.prefix[:lcp])
			splitNode.parent = node.parent
			splitNode.charInParent = node.charInParent
			if splitNode.parent != nil {
				for i := range splitNode.parent.children {
					if splitNode.parent.children[i].b == splitNode.charInParent {
						splitNode.parent.children[i].n = splitNode
						break
					}
				}
			}

			// Update the existing node
			node.prefix = node.prefix[lcp:]
			node.parent = splitNode
			node.charInParent = node.prefix[0]
			splitNode.children = append(splitNode.children, trieChild{node.charInParent, node})

			// Add new node for the remaining part if there is any
			if lcp < len(remainingKey) {
				newNode := newTrieNode(remainingKey[lcp:])
				newNode.val = val
				newNode.hasVal = true
				newNode.parent = splitNode
				newNode.charInParent = remainingKey[lcp]
				splitNode.children = append(splitNode.children, trieChild{newNode.charInParent, newNode})
				targetNode = newNode
			} else {
				splitNode.val = val
				splitNode.hasVal = true
				targetNode = splitNode
			}
		}
	}

	sc.currentSize += val.Size()
	if targetNode.lruElem != nil {
		sc.entries.MoveToFront(targetNode.lruElem)
		targetNode.lruElem.Value = targetNode
	} else {
		targetNode.lruElem = sc.entries.PushFront(targetNode)
	}

	sc.evictIfNeeded()
}

func (sc *statCacheTrie) pruneNode(node *trieNode) {
	for node != nil && node != sc.root && !node.hasVal && len(node.children) == 0 {
		parent := node.parent
		if parent != nil {
			for i := range parent.children {
				if parent.children[i].b == node.charInParent {
					parent.children = append(parent.children[:i], parent.children[i+1:]...)
					break
				}
			}
			node.parent = nil // Help GC
		}
		node = parent
	}
}

func (sc *statCacheTrie) evictIfNeeded() {
	for sc.currentSize > sc.maxSize && sc.entries.Len() > 0 {
		elem := sc.entries.Back()
		node := elem.Value.(*trieNode)

		sc.currentSize -= node.val.Size()
		node.hasVal = false
		node.val = entry{}
		node.lruElem = nil
		sc.entries.Remove(elem)

		sc.pruneNode(node)
	}
}

func (sc *statCacheTrie) Insert(m *gcs.MinObject, expiration time.Time) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	name := m.Name
	node, _, exactMatch := sc.findNode(name)

	if exactMatch && node.hasVal {
		if !shouldReplace(m, node.val) {
			return
		}
	}

	mCopy := *m
	mCopy.Name = ""

	e := entry{
		m:          &mCopy,
		expiration: expiration,
		key:        name,
	}

	sc.insertNode(name, e)
}

func (sc *statCacheTrie) InsertImplicitDir(objectName string, expiration time.Time) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	node, _, exactMatch := sc.findNode(objectName)

	if exactMatch && node.hasVal && node.val.m != nil {
		return
	}

	e := entry{
		implicitDir: true,
		expiration:  expiration,
		key:         objectName,
	}

	sc.insertNode(objectName, e)
}

func (sc *statCacheTrie) AddNegativeEntry(name string, expiration time.Time) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	e := entry{
		m:          nil,
		expiration: expiration,
		key:        name,
	}

	sc.insertNode(name, e)
}

func (sc *statCacheTrie) AddNegativeEntryForFolder(folderName string, expiration time.Time) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	e := entry{
		f:          nil,
		expiration: expiration,
		key:        folderName,
	}

	sc.insertNode(folderName, e)
}

func (sc *statCacheTrie) eraseNode(name string) {
	node, _, exactMatch := sc.findNode(name)
	if exactMatch && node.hasVal {
		sc.currentSize -= node.val.Size()
		if node.lruElem != nil {
			sc.entries.Remove(node.lruElem)
			node.lruElem = nil
		}
		node.hasVal = false
		node.val = entry{}
		sc.pruneNode(node)
	}
}

func (sc *statCacheTrie) Erase(name string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.eraseNode(name)
}

func (sc *statCacheTrie) LookUp(name string, now time.Time) (bool, *gcs.MinObject) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	node, _, exactMatch := sc.findNode(name)
	if !exactMatch || !node.hasVal {
		return false, nil
	}

	if node.val.expiration.Before(now) {
		sc.eraseNode(name)
		return false, nil
	}

	if node.lruElem != nil {
		sc.entries.MoveToFront(node.lruElem)
	}

	if node.val.implicitDir {
		return true, &gcs.MinObject{Name: name}
	}

	if node.val.m != nil {
		mCopy := *node.val.m
		mCopy.Name = name
		return true, &mCopy
	}

	return true, nil
}

func (sc *statCacheTrie) InsertFolder(f *gcs.Folder, expiration time.Time) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	fCopy := *f
	fCopy.Name = ""

	e := entry{
		f:          &fCopy,
		expiration: expiration,
		key:        f.Name,
	}

	sc.insertNode(f.Name, e)
}

func (sc *statCacheTrie) LookUpFolder(folderName string, now time.Time) (bool, *gcs.Folder) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	node, _, exactMatch := sc.findNode(folderName)
	if !exactMatch || !node.hasVal {
		return false, nil
	}

	if node.val.expiration.Before(now) {
		sc.eraseNode(folderName)
		return false, nil
	}

	if node.lruElem != nil {
		sc.entries.MoveToFront(node.lruElem)
	}

	if node.val.f != nil {
		fCopy := *node.val.f
		fCopy.Name = folderName
		return true, &fCopy
	}

	return true, nil
}

func (sc *statCacheTrie) dfsErase(node *trieNode) {
	if node.hasVal {
		sc.currentSize -= node.val.Size()
		if node.lruElem != nil {
			sc.entries.Remove(node.lruElem)
			node.lruElem = nil
		}
		node.hasVal = false
		node.val = entry{}
	}
	for _, child := range node.children {
		sc.dfsErase(child.n)
	}
	// Clear children to free memory
	node.children = nil
	// Now try pruning this node (it has no children and no value)
	sc.pruneNode(node)
}

func (sc *statCacheTrie) EraseEntriesWithGivenPrefix(prefix string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if prefix == "" {
		sc.dfsErase(sc.root)
		return
	}

	node := sc.root
	remainingPrefix := prefix

	for len(remainingPrefix) > 0 {
		c := remainingPrefix[0]
		var child *trieNode
		for i := range node.children {
			if node.children[i].b == c {
				child = node.children[i].n
				break
			}
		}
		if child == nil {
			return // Prefix not found
		}

		lcp := longestCommonPrefix(child.prefix, remainingPrefix)
		if lcp == len(remainingPrefix) {
			// Found the subtree matching the prefix
			sc.dfsErase(child)
			for i := range node.children {
				if node.children[i].b == c {
					node.children = append(node.children[:i], node.children[i+1:]...)
					break
				}
			}
			return
		} else if lcp == len(child.prefix) {
			// Prefix continues down this path
			remainingPrefix = remainingPrefix[lcp:]
			node = child
		} else {
			// Prefix diverges
			return
		}
	}
}

// NewStatCacheTrieBucketView creates a new bucket-view to the passed shared-cache object.
func NewStatCacheTrieBucketView(sc StatCache, bn string) StatCache {
	return &statCacheTrieBucketView{
		sharedCache: sc,
		bucketName:  bn,
	}
}

// statCacheTrieBucketView is a special type of StatCache which
// shares its underlying cache object with other
// statCacheTrieBucketView objects.
type statCacheTrieBucketView struct {
	sharedCache StatCache
	bucketName  string
}

func (sc *statCacheTrieBucketView) key(objectName string) string {
	if sc.bucketName != "" {
		return sc.bucketName + "/" + objectName
	}
	return objectName
}

func (sc *statCacheTrieBucketView) Insert(m *gcs.MinObject, expiration time.Time) {
	clone := *m
	clone.Name = sc.key(m.Name)
	sc.sharedCache.Insert(&clone, expiration)
}

func (sc *statCacheTrieBucketView) InsertImplicitDir(objectName string, expiration time.Time) {
	name := sc.key(objectName)
	sc.sharedCache.InsertImplicitDir(name, expiration)
}

func (sc *statCacheTrieBucketView) AddNegativeEntry(objectName string, expiration time.Time) {
	name := sc.key(objectName)
	sc.sharedCache.AddNegativeEntry(name, expiration)
}

func (sc *statCacheTrieBucketView) AddNegativeEntryForFolder(folderName string, expiration time.Time) {
	name := sc.key(folderName)
	sc.sharedCache.AddNegativeEntryForFolder(name, expiration)
}

func (sc *statCacheTrieBucketView) Erase(objectName string) {
	name := sc.key(objectName)
	sc.sharedCache.Erase(name)
}

func (sc *statCacheTrieBucketView) LookUp(objectName string, now time.Time) (bool, *gcs.MinObject) {
	name := sc.key(objectName)
	hit, m := sc.sharedCache.LookUp(name, now)
	if hit && m != nil {
		clone := *m
		clone.Name = objectName
		return hit, &clone
	}
	return hit, m
}

func (sc *statCacheTrieBucketView) InsertFolder(f *gcs.Folder, expiration time.Time) {
	clone := *f
	clone.Name = sc.key(f.Name)
	sc.sharedCache.InsertFolder(&clone, expiration)
}

func (sc *statCacheTrieBucketView) LookUpFolder(folderName string, now time.Time) (bool, *gcs.Folder) {
	name := sc.key(folderName)
	hit, f := sc.sharedCache.LookUpFolder(name, now)
	if hit && f != nil {
		clone := *f
		clone.Name = folderName
		return hit, &clone
	}
	return hit, f
}

func (sc *statCacheTrieBucketView) EraseEntriesWithGivenPrefix(prefix string) {
	name := sc.key(prefix)
	sc.sharedCache.EraseEntriesWithGivenPrefix(name)
}
