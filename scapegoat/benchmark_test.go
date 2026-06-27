package scapegoat

import "testing"

func BenchmarkInsert(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tree, _ := NewOrdered[int, int](2.0 / 3.0)
		for key := 0; key < 1000; key++ {
			tree.Insert(key, key)
		}
	}
}

func BenchmarkSearch(b *testing.B) {
	tree, _ := NewOrdered[int, int](2.0 / 3.0)
	for key := 0; key < 100000; key++ {
		tree.Insert(key, key)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.Search(i % 100000)
	}
}

func BenchmarkDelete(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tree, _ := NewOrdered[int, int](2.0 / 3.0)
		for key := 0; key < 1000; key++ {
			tree.Insert(key, key)
		}
		for key := 0; key < 1000; key++ {
			tree.Delete(key)
		}
	}
}
