package lru

import "container/list"

// Index represents the key-value index used by the LRU Cache.
type Index interface {
	insert(key string, element *list.Element)
	lookup(key string) (*list.Element, bool)
	delete(key string) *list.Element
	erasePrefix(prefix string) []*list.Element
	len() int
}

// mapIndex is the traditional map-based index.
type mapIndex struct {
	m map[string]*list.Element
}

func newMapIndex() *mapIndex {
	return &mapIndex{m: make(map[string]*list.Element)}
}

func (mi *mapIndex) insert(key string, element *list.Element) {
	mi.m[key] = element
}

func (mi *mapIndex) lookup(key string) (*list.Element, bool) {
	el, ok := mi.m[key]
	return el, ok
}

func (mi *mapIndex) delete(key string) *list.Element {
	el := mi.m[key]
	delete(mi.m, key)
	return el
}

func (mi *mapIndex) erasePrefix(prefix string) []*list.Element {
	var deleted []*list.Element
	// To match prefix, we unfortunately have to iterate the whole map
	// which is why the trie approach is better.
	for k, el := range mi.m {
		// Only check if it starts with the prefix or is exact match (depends on Go strings.HasPrefix)
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			deleted = append(deleted, el)
			delete(mi.m, k)
		}
	}
	return deleted
}

func (mi *mapIndex) len() int {
	return len(mi.m)
}
