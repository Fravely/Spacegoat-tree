package scapegoat

import "testing"

func mustIntTree(t *testing.T, alpha float64) *Tree[int, string] {
	t.Helper()
	tree, err := NewOrdered[int, string](alpha)
	if err != nil {
		t.Fatal(err)
	}
	return tree
}
