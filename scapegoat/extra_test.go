package scapegoat

import (
	"math/rand"
	"strconv"
	"testing"
)

func TestSortedInsertTriggersRebuild(t *testing.T) {
	tree := mustIntTree(t, 2.0/3.0)
	for key := 1; key <= 1000; key++ {
		tree.Insert(key, "")
	}
	stats := tree.Stats()
	if stats.Rebuilds == 0 {
		t.Fatal("sorted insertion should trigger at least one rebuild")
	}
	maxAllowed := tree.maxDepth(stats.Size)
	if stats.Height > maxAllowed+2 {
		t.Fatalf("Height = %d exceeds reasonable bound %d", stats.Height, maxAllowed+2)
	}
}

func TestDeleteCasesAndGlobalRebuild(t *testing.T) {
	tree := mustIntTree(t, 2.0/3.0)
	for key := 1; key <= 200; key++ {
		tree.Insert(key, "")
	}
	before := tree.Stats()

	for key := 1; key <= 100; key++ {
		if !tree.Delete(key) {
			t.Fatalf("Delete(%d) should succeed", key)
		}
	}
	after := tree.Stats()

	if after.Size != 100 {
		t.Fatalf("Size = %d, want 100", after.Size)
	}
	if after.Rebuilds <= before.Rebuilds {
		t.Fatal("deleting 50% should trigger global rebuild")
	}
}

func TestRandomOperationsMatchMap(t *testing.T) {
	tree := mustIntTree(t, 2.0/3.0)
	ref := make(map[int]string)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 5000; i++ {
		op := rng.Intn(3)
		key := rng.Intn(500)
		switch op {
		case 0:
			val := strconv.Itoa(rng.Int())
			inserted := tree.Insert(key, val)
			if _, exists := ref[key]; exists {
				if inserted {
					t.Fatalf("key %d should update, not insert", key)
				}
				ref[key] = val
			} else {
				if !inserted {
					t.Fatalf("key %d should insert", key)
				}
				ref[key] = val
			}
		case 1:
			got, ok := tree.Search(key)
			want, exists := ref[key]
			if ok != exists {
				t.Fatalf("Search(%d) presence mismatch: got %v want %v", key, ok, exists)
			}
			if exists && got != want {
				t.Fatalf("Search(%d) = %q, want %q", key, got, want)
			}
		case 2:
			deleted := tree.Delete(key)
			_, exists := ref[key]
			if deleted != exists {
				t.Fatalf("Delete(%d) = %v, want %v", key, deleted, exists)
			}
			if deleted {
				delete(ref, key)
			}
		}
	}

	if tree.Len() != len(ref) {
		t.Fatalf("Len() = %d, map size = %d", tree.Len(), len(ref))
	}
}

func TestNewRejectsInvalidAlpha(t *testing.T) {
	if _, err := NewOrdered[int, string](0.5); err == nil {
		t.Fatal("alpha=0.5 should be rejected")
	}
	if _, err := NewOrdered[int, string](1.0); err == nil {
		t.Fatal("alpha=1.0 should be rejected")
	}
}
