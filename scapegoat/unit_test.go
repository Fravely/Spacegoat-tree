package scapegoat

import (
	"reflect"
	"testing"
)

func TestEmptyTree(t *testing.T) {
	tree := mustIntTree(t, 2.0/3.0)

	if tree.Len() != 0 {
		t.Fatalf("Len() = %d, want 0", tree.Len())
	}
	if _, ok := tree.Search(1); ok {
		t.Fatal("Search on empty tree should return false")
	}
	if tree.Delete(1) {
		t.Fatal("Delete on empty tree should return false")
	}
	if entries := tree.InOrder(); len(entries) != 0 {
		t.Fatalf("InOrder() = %v, want empty slice", entries)
	}
	if tree.Snapshot() != nil {
		t.Fatal("Snapshot() of empty tree should be nil")
	}
	if tree.Height() != -1 {
		t.Fatalf("Height() = %d, want -1", tree.Height())
	}

	stats := tree.Stats()
	if stats.Size != 0 || stats.MaxSize != 0 || stats.Height != -1 ||
		stats.Rebuilds != 0 || stats.RebuiltNodes != 0 {
		t.Fatalf("Stats() = %+v, want zero values with Height -1", stats)
	}
}

func TestInsertWithTraceOnEmptyTree(t *testing.T) {
	tree := mustIntTree(t, 2.0/3.0)

	inserted, trace := tree.InsertWithTrace(10, "root")
	if !inserted {
		t.Fatal("first insertion should return inserted=true")
	}
	if trace.Updated {
		t.Fatal("first insertion should not be an update")
	}
	if !trace.Inserted {
		t.Fatal("trace.Inserted should be true")
	}
	if trace.Depth != 0 {
		t.Fatalf("trace.Depth = %d, want 0", trace.Depth)
	}
	if trace.MaxDepth != 0 {
		t.Fatalf("trace.MaxDepth = %d, want 0", trace.MaxDepth)
	}
	if !reflect.DeepEqual(trace.Path, []int{10}) {
		t.Fatalf("trace.Path = %v, want [10]", trace.Path)
	}
	if trace.Scapegoat != nil {
		t.Fatal("empty-tree insertion should not trigger a scapegoat rebuild")
	}
}

func TestInsertWithTraceUpdateExistingKey(t *testing.T) {
	tree := mustIntTree(t, 2.0/3.0)
	tree.Insert(5, "old")

	inserted, trace := tree.InsertWithTrace(5, "new")
	if inserted {
		t.Fatal("updating an existing key should return inserted=false")
	}
	if !trace.Updated {
		t.Fatal("trace.Updated should be true")
	}
	if trace.Inserted {
		t.Fatal("trace.Inserted should be false on update")
	}
	if trace.Depth != 0 {
		t.Fatalf("trace.Depth = %d, want 0", trace.Depth)
	}
	if !reflect.DeepEqual(trace.Path, []int{5}) {
		t.Fatalf("trace.Path = %v, want [5]", trace.Path)
	}

	value, ok := tree.Search(5)
	if !ok || value != "new" {
		t.Fatalf("Search(5) = %q, %v", value, ok)
	}
}

func TestInsertWithTraceRecordsPathAndDepth(t *testing.T) {
	tree := mustIntTree(t, 2.0/3.0)
	for _, key := range []int{10, 5, 15, 3} {
		tree.Insert(key, "")
	}

	_, trace := tree.InsertWithTrace(7, "leaf")
	if !trace.Inserted {
		t.Fatal("trace.Inserted should be true for a new key")
	}
	if trace.Updated {
		t.Fatal("trace.Updated should be false for a new key")
	}
	if !reflect.DeepEqual(trace.Path, []int{10, 5, 7}) {
		t.Fatalf("trace.Path = %v, want [10 5 7]", trace.Path)
	}
	if trace.Depth != 2 {
		t.Fatalf("trace.Depth = %d, want 2", trace.Depth)
	}
}

func TestInsertWithTraceTriggersScapegoatRebuild(t *testing.T) {
	tree := mustIntTree(t, 2.0/3.0)

	var sawScapegoat bool
	for key := 1; key <= 200; key++ {
		_, trace := tree.InsertWithTrace(key, "")
		if trace.Scapegoat != nil {
			sawScapegoat = true
			if len(trace.RebuiltKeys) == 0 {
				t.Fatal("scapegoat rebuild should record rebuilt keys")
			}
			break
		}
	}
	if !sawScapegoat {
		t.Fatal("sorted insertion should eventually select a scapegoat node")
	}
}

func TestSnapshotPreservesTreeShape(t *testing.T) {
	tree := mustIntTree(t, 2.0/3.0)
	for _, key := range []int{8, 3, 10, 1, 6} {
		tree.Insert(key, "")
	}

	got := tree.Snapshot()
	want := &NodeSnapshot[int, string]{
		Key:   8,
		Value: "",
		Left: &NodeSnapshot[int, string]{
			Key:   3,
			Value: "",
			Left:  &NodeSnapshot[int, string]{Key: 1, Value: ""},
			Right: &NodeSnapshot[int, string]{Key: 6, Value: ""},
		},
		Right: &NodeSnapshot[int, string]{Key: 10, Value: ""},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Snapshot() mismatch:\ngot  %+v\nwant %+v", got, want)
	}
}

func TestHeightForKnownShapes(t *testing.T) {
	tree := mustIntTree(t, 2.0/3.0)
	if tree.Height() != -1 {
		t.Fatalf("empty Height() = %d, want -1", tree.Height())
	}

	tree.Insert(1, "")
	if tree.Height() != 0 {
		t.Fatalf("single-node Height() = %d, want 0", tree.Height())
	}

	tree.Insert(2, "")
	tree.Insert(3, "")
	if tree.Height() != 2 {
		t.Fatalf("degenerate chain Height() = %d, want 2", tree.Height())
	}
}

func TestDeleteNodeWithTwoChildren(t *testing.T) {
	tree := mustIntTree(t, 2.0/3.0)
	for _, key := range []int{10, 5, 15, 3, 7, 12, 18} {
		tree.Insert(key, "")
	}

	if !tree.Delete(10) {
		t.Fatal("Delete(10) should succeed")
	}
	if _, ok := tree.Search(10); ok {
		t.Fatal("deleted root should not be found")
	}

	entries := tree.InOrder()
	for i := 1; i < len(entries); i++ {
		if entries[i-1].Key >= entries[i].Key {
			t.Fatalf("traversal is not ordered after delete: %v", entries)
		}
	}
	if len(entries) != 6 {
		t.Fatalf("Len after delete = %d, want 6", tree.Len())
	}
}

func TestDeleteLastElementResetsTree(t *testing.T) {
	tree := mustIntTree(t, 2.0/3.0)
	tree.Insert(42, "only")

	if !tree.Delete(42) {
		t.Fatal("Delete(42) should succeed")
	}
	if tree.Len() != 0 {
		t.Fatalf("Len() = %d, want 0", tree.Len())
	}
	if tree.Snapshot() != nil {
		t.Fatal("tree should be empty after deleting its only element")
	}
	if stats := tree.Stats(); stats.MaxSize != 0 {
		t.Fatalf("MaxSize = %d, want 0 after full delete", stats.MaxSize)
	}
}

func TestStringKeysWithNewOrdered(t *testing.T) {
	tree, err := NewOrdered[string, int](0.75)
	if err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"lima", "arequipa", "cusco", "piura"} {
		if !tree.Insert(key, len(key)) {
			t.Fatalf("key %q unexpectedly existed", key)
		}
	}

	value, ok := tree.Search("cusco")
	if !ok || value != 5 {
		t.Fatalf("Search(cusco) = %d, %v", value, ok)
	}

	entries := tree.InOrder()
	want := []string{"arequipa", "cusco", "lima", "piura"}
	for i, key := range want {
		if entries[i].Key != key {
			t.Fatalf("entries[%d].Key = %q, want %q", i, entries[i].Key, key)
		}
	}
}

func TestCustomComparator(t *testing.T) {
	type item struct {
		priority int
		label    string
	}

	tree, err := New[item, string](0.7, func(a, b item) bool { return a.priority < b.priority })
	if err != nil {
		t.Fatal(err)
	}

	a := item{priority: 1, label: "low"}
	b := item{priority: 5, label: "high"}
	c := item{priority: 3, label: "mid"}

	tree.Insert(b, "B")
	tree.Insert(a, "A")
	tree.Insert(c, "C")

	value, ok := tree.Search(item{priority: 5, label: "ignored"})
	if !ok || value != "B" {
		t.Fatalf("Search high priority = %q, %v", value, ok)
	}

	entries := tree.InOrder()
	if entries[0].Key.priority != 1 || entries[2].Key.priority != 5 {
		t.Fatalf("custom order failed: %+v", entries)
	}
}

func TestStatsReflectOperations(t *testing.T) {
	tree := mustIntTree(t, 2.0/3.0)

	for key := 1; key <= 500; key++ {
		tree.Insert(key, "")
	}
	beforeDelete := tree.Stats()

	for key := 1; key <= 400; key++ {
		tree.Delete(key)
	}
	afterDelete := tree.Stats()

	if afterDelete.Size != 100 {
		t.Fatalf("Size = %d, want 100", afterDelete.Size)
	}
	if afterDelete.Rebuilds <= beforeDelete.Rebuilds {
		t.Fatal("mass deletion should increase rebuild count")
	}
	if afterDelete.RebuiltNodes <= beforeDelete.RebuiltNodes {
		t.Fatal("mass deletion should increase rebuilt node count")
	}
}
