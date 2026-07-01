package scapegoat

import (
	"fmt"
	"math/rand"
	"testing"
)

const benchAlpha = 2.0 / 3.0

// Tamaños de prueba en miles de nodos (1K … 50K).
var benchSizes = []int{1_000, 5_000, 10_000, 50_000}

func newBenchTree(b *testing.B) *Tree[int, struct{}] {
	b.Helper()
	tree, err := NewOrdered[int, struct{}](benchAlpha)
	if err != nil {
		b.Fatal(err)
	}
	return tree
}

func benchKeysRandom(n int, seed int64) []int {
	rng := rand.New(rand.NewSource(seed))
	keys := make([]int, n)
	seen := make(map[int]struct{}, n)
	for i := 0; i < n; {
		k := rng.Intn(n * 4)
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		keys[i] = k
		i++
	}
	return keys
}

func benchKeysSorted(n int) []int {
	keys := make([]int, n)
	for i := range keys {
		keys[i] = i
	}
	return keys
}

// BenchmarkInsert mide inserción secuencial ordenada (0..n-1) a distintas escalas.
func BenchmarkInsert(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(fmt.Sprintf("sorted_%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tree := newBenchTree(b)
				for k := 0; k < n; k++ {
					tree.Insert(k, struct{}{})
				}
			}
		})
	}
}

// BenchmarkInsertRandom mide inserción de claves aleatorias únicas.
func BenchmarkInsertRandom(b *testing.B) {
	for _, n := range benchSizes {
		keys := benchKeysRandom(n, 42)
		b.Run(fmt.Sprintf("random_%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tree := newBenchTree(b)
				for _, k := range keys {
					tree.Insert(k, struct{}{})
				}
			}
		})
	}
}

// BenchmarkSearch mide búsqueda en árboles precargados con n nodos.
func BenchmarkSearch(b *testing.B) {
	for _, n := range append(benchSizes, 100_000) {
		n := n
		tree := newBenchTree(b)
		for k := 0; k < n; k++ {
			tree.Insert(k, struct{}{})
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tree.Search(i % n)
			}
		})
	}
}

// BenchmarkDelete mide inserción + eliminación completa a distintas escalas.
func BenchmarkDelete(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tree := newBenchTree(b)
				for k := 0; k < n; k++ {
					tree.Insert(k, struct{}{})
				}
				for k := 0; k < n; k++ {
					tree.Delete(k)
				}
			}
		})
	}
}

// BenchmarkInOrder mide recorrido in-order sobre árboles de miles de nodos.
func BenchmarkInOrder(b *testing.B) {
	for _, n := range benchSizes {
		tree := newBenchTree(b)
		for k := 0; k < n; k++ {
			tree.Insert(k, struct{}{})
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = tree.InOrder()
			}
		})
	}
}

// BenchmarkRebuilds compara inserción ordenada vs aleatoria (miles de nodos).
func BenchmarkRebuilds(b *testing.B) {
	for _, n := range []int{1_000, 5_000, 10_000} {
		n := n
		b.Run(fmt.Sprintf("sorted_%d", n), func(b *testing.B) {
			keys := benchKeysSorted(n)
			for i := 0; i < b.N; i++ {
				tree := newBenchTree(b)
				for _, k := range keys {
					tree.Insert(k, struct{}{})
				}
			}
		})
		keys := benchKeysRandom(n, 99)
		b.Run(fmt.Sprintf("random_%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tree := newBenchTree(b)
				for _, k := range keys {
					tree.Insert(k, struct{}{})
				}
			}
		})
	}
}

// BenchmarkMixedWorkload simula carga mixta sobre 10K nodos: 70% búsqueda, 20% inserción, 10% eliminación.
func BenchmarkMixedWorkload(b *testing.B) {
	const n = 10_000
	tree := newBenchTree(b)
	for k := 0; k < n; k++ {
		tree.Insert(k, struct{}{})
	}
	rng := rand.New(rand.NewSource(456))
	nextInsert := n

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		switch rng.Intn(10) {
		case 0:
			tree.Delete(rng.Intn(n))
		case 1, 2:
			tree.Insert(nextInsert, struct{}{})
			nextInsert++
		default:
			tree.Search(rng.Intn(n))
		}
	}
}
