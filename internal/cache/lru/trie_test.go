package lru

import (
	"container/list"
	"testing"
)

func TestPathTrie(t *testing.T) {
	trie := newPathTrie()
	if trie.len() != 0 {
		t.Fatalf("Expected len 0")
	}

	elA := &list.Element{}
	trie.insert("a", elA)
	if trie.len() != 1 {
		t.Fatalf("Expected len 1, got %d", trie.len())
	}

	elAB := &list.Element{}
	trie.insert("ab", elAB)
	if trie.len() != 2 {
		t.Fatalf("Expected len 2")
	}

	elABC := &list.Element{}
	trie.insert("abc", elABC)
	if trie.len() != 3 {
		t.Fatalf("Expected len 3")
	}

	elBCD := &list.Element{}
	trie.insert("bcd", elBCD)
	if trie.len() != 4 {
		t.Fatalf("Expected len 4")
	}

	if val, ok := trie.lookup("ab"); !ok || val != elAB {
		t.Fatalf("Expected to find ab")
	}

	if val, ok := trie.lookup("abc"); !ok || val != elABC {
		t.Fatalf("Expected to find abc")
	}

	if _, ok := trie.lookup("xyz"); ok {
		t.Fatalf("Should not find xyz")
	}

	deleted := trie.erasePrefix("ab")
	if len(deleted) != 2 {
		t.Fatalf("Expected to delete 2 elements, got %d", len(deleted))
	}

	if trie.len() != 2 {
		t.Fatalf("Expected len 2, got %d", trie.len())
	}

	if _, ok := trie.lookup("ab"); ok {
		t.Fatalf("ab should be deleted")
	}
	if _, ok := trie.lookup("abc"); ok {
		t.Fatalf("abc should be deleted")
	}
	if val, ok := trie.lookup("a"); !ok || val != elA {
		t.Fatalf("a should remain")
	}

	elDeleteA := trie.delete("a")
	if elDeleteA != elA {
		t.Fatalf("Expected to delete a")
	}
	if trie.len() != 1 {
		t.Fatalf("Expected len 1")
	}
}
