package scapegoat

import (
	"math/rand"
	"sort"
	"testing"
)

func mustIntTree(t *testing.T, alpha float64) *Tree[int, string] {
	t.Helper()
	tree, err := NewOrdered[int, string](alpha)
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func TestInsertSearchUpdateAndOrder(t *testing.T) {
	tree := mustIntTree(t, 2.0/3.0)
	for _, key := range []int{8, 3, 10, 1, 6, 14, 4, 7, 13} {
		if !tree.Insert(key, "value") {
			t.Fatalf("key %d unexpectedly existed", key)
		}
	}
	if tree.Insert(6, "updated") {
		t.Fatal("updating an existing key must return false")
	}
	value, ok := tree.Search(6)
	if !ok || value != "updated" {
		t.Fatalf("Search(6) = %q, %v", value, ok)
	}
	if _, ok := tree.Search(99); ok {
		t.Fatal("unexpected key 99")
	}

	entries := tree.InOrder()
	for i := 1; i < len(entries); i++ {
		if entries[i-1].Key >= entries[i].Key {
			t.Fatalf("traversal is not ordered: %v", entries)
		}
	}
}

func TestSortedInsertTriggersRebuild(t *testing.T) {
	tree := mustIntTree(t, 2.0/3.0)
	for key := 1; key <= 1000; key++ {
		tree.Insert(key, "")
	}
	stats := tree.Stats()
	if stats.Rebuilds == 0 {
		t.Fatal("sorted insertion should trigger subtree rebuilds")
	}
	if stats.Height > tree.maxDepth(stats.MaxSize)+1 {
		t.Fatalf("height %d is unexpectedly high", stats.Height)
	}
}

func TestDeleteCasesAndGlobalRebuild(t *testing.T) {
	tree := mustIntTree(t, 0.6)
	for key := 1; key <= 100; key++ {
		tree.Insert(key, "")
	}
	before := tree.Stats().Rebuilds
	for key := 1; key <= 50; key++ {
		if !tree.Delete(key) {
			t.Fatalf("could not delete %d", key)
		}
	}
	if tree.Delete(1000) {
		t.Fatal("deleting a missing key must return false")
	}
	if tree.Len() != 50 {
		t.Fatalf("Len() = %d, want 50", tree.Len())
	}
	if tree.Stats().Rebuilds <= before {
		t.Fatal("enough deletions should trigger a full rebuild")
	}
	for key := 51; key <= 100; key++ {
		if _, ok := tree.Search(key); !ok {
			t.Fatalf("remaining key %d not found", key)
		}
	}
}

func TestRandomOperationsMatchMap(t *testing.T) {
	tree, err := NewOrdered[int, int](0.7)
	if err != nil {
		t.Fatal(err)
	}
	want := map[int]int{}
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 5000; i++ {
		key := rng.Intn(500)
		if rng.Intn(3) != 0 {
			tree.Insert(key, i)
			want[key] = i
		} else {
			got := tree.Delete(key)
			_, existed := want[key]
			if got != existed {
				t.Fatalf("Delete(%d) = %v, want %v", key, got, existed)
			}
			delete(want, key)
		}
	}

	entries := tree.InOrder()
	keys := make([]int, 0, len(want))
	for key := range want {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	if len(entries) != len(keys) {
		t.Fatalf("got %d entries, want %d", len(entries), len(keys))
	}
	for i, key := range keys {
		if entries[i].Key != key || entries[i].Value != want[key] {
			t.Fatalf("entry %d = %+v", i, entries[i])
		}
	}
}

func TestInvalidConfiguration(t *testing.T) {
	for _, alpha := range []float64{0.5, 1, 0, 2} {
		if _, err := NewOrdered[int, int](alpha); err == nil {
			t.Fatalf("NewOrdered(%v) should fail", alpha)
		}
	}
	if _, err := New[int, int](0.7, nil); err == nil {
		t.Fatal("nil comparator should fail")
	}
}
