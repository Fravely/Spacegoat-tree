// Package scapegoat implements a Scapegoat Tree, a binary search tree that
// restores balance by rebuilding unbalanced subtrees instead of using rotations.
package scapegoat

import (
	"fmt"
	"math"
)

// Ordered contains the built-in types that support the < operator.
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 | ~string
}

// Entry is a key-value pair returned by an ordered traversal.
type Entry[K, V any] struct {
	Key   K `json:"key"`
	Value V `json:"value"`
}

// NodeSnapshot exposes the tree shape for visualization without exposing the
// internal node pointers.
type NodeSnapshot[K, V any] struct {
	Key   K                   `json:"key"`
	Value V                   `json:"value"`
	Left  *NodeSnapshot[K, V] `json:"left,omitempty"`
	Right *NodeSnapshot[K, V] `json:"right,omitempty"`
}

// Stats summarizes the current state and the rebuild work performed.
type Stats struct {
	Size         int `json:"size"`
	MaxSize      int `json:"maxSize"`
	Height       int `json:"height"`
	Rebuilds     int `json:"rebuilds"`
	RebuiltNodes int `json:"rebuiltNodes"`
}

// InsertTrace describes the decisions made during one insertion. It is useful
// for visualizations and keeps the tree nodes independent from UI state.
type InsertTrace[K any] struct {
	Path        []K  `json:"path"`
	Inserted    bool `json:"inserted"`
	Updated     bool `json:"updated"`
	Depth       int  `json:"depth"`
	MaxDepth    int  `json:"maxDepth"`
	Scapegoat   *K   `json:"scapegoat,omitempty"`
	RebuiltKeys []K  `json:"rebuiltKeys,omitempty"`
}

type node[K, V any] struct {
	key         K
	value       V
	left, right *node[K, V]
}

// Tree is a Scapegoat Tree. Alpha must be strictly between 0.5 and 1.
// Smaller alpha values produce stricter balance and more rebuilds.
type Tree[K, V any] struct {
	root         *node[K, V]
	less         func(K, K) bool
	alpha        float64
	n, q         int
	rebuilds     int
	rebuiltNodes int
}

// New creates a tree using less as the strict ordering relation.
func New[K, V any](alpha float64, less func(K, K) bool) (*Tree[K, V], error) {
	if alpha <= 0.5 || alpha >= 1 {
		return nil, fmt.Errorf("alpha must be strictly between 0.5 and 1")
	}
	if less == nil {
		return nil, fmt.Errorf("less comparator cannot be nil")
	}
	return &Tree[K, V]{alpha: alpha, less: less}, nil
}

// NewOrdered creates a tree for a built-in ordered key type.
func NewOrdered[K Ordered, V any](alpha float64) (*Tree[K, V], error) {
	return New[K, V](alpha, func(a, b K) bool { return a < b })
}

// Len returns the number of entries in the tree.
func (t *Tree[K, V]) Len() int { return t.n }

func (t *Tree[K, V]) equal(a, b K) bool {
	return !t.less(a, b) && !t.less(b, a)
}

// Search looks up key and reports whether it was found.
func (t *Tree[K, V]) Search(key K) (V, bool) {
	for current := t.root; current != nil; {
		if t.equal(key, current.key) {
			return current.value, true
		}
		if t.less(key, current.key) {
			current = current.left
		} else {
			current = current.right
		}
	}
	var zero V
	return zero, false
}

// Insert adds a key-value pair. If the key exists, it updates the value and
// returns false; otherwise it returns true.
func (t *Tree[K, V]) Insert(key K, value V) bool {
	inserted, _ := t.insert(key, value)
	return inserted
}

// InsertWithTrace adds or updates a key-value pair and returns the insertion
// decisions needed to explain the Scapegoat Tree algorithm step by step.
func (t *Tree[K, V]) InsertWithTrace(key K, value V) (bool, InsertTrace[K]) {
	return t.insert(key, value)
}

func (t *Tree[K, V]) insert(key K, value V) (bool, InsertTrace[K]) {
	trace := InsertTrace[K]{Depth: -1, MaxDepth: t.maxDepth(t.q)}

	if t.root == nil {
		t.root = &node[K, V]{key: key, value: value}
		t.n, t.q = 1, 1
		trace.Path = append(trace.Path, key)
		trace.Inserted = true
		trace.Depth = 0
		trace.MaxDepth = 0
		return true, trace
	}

	path := make([]*node[K, V], 0, t.maxDepth(t.q)+2)
	current := t.root
	for {
		path = append(path, current)
		trace.Path = append(trace.Path, current.key)
		if t.equal(key, current.key) {
			current.value = value
			trace.Updated = true
			trace.Depth = len(path) - 1
			return false, trace
		}
		if t.less(key, current.key) {
			if current.left == nil {
				current.left = &node[K, V]{key: key, value: value}
				path = append(path, current.left)
				trace.Path = append(trace.Path, current.left.key)
				break
			}
			current = current.left
		} else {
			if current.right == nil {
				current.right = &node[K, V]{key: key, value: value}
				path = append(path, current.right)
				trace.Path = append(trace.Path, current.right.key)
				break
			}
			current = current.right
		}
	}

	t.n++
	t.q++

	depth := len(path) - 1
	trace.Inserted = true
	trace.Depth = depth
	trace.MaxDepth = t.maxDepth(t.q)
	if depth > t.maxDepth(t.q) {
		for i := len(path) - 2; i >= 0; i-- {
			parent, child := path[i], path[i+1]
			if float64(size(child)) > t.alpha*float64(size(parent)) {
				scapegoatKey := parent.key
				trace.Scapegoat = &scapegoatKey
				flattenKeys(parent, &trace.RebuiltKeys)
				rebuilt := t.rebuild(parent)
				if i == 0 {
					t.root = rebuilt
				} else if path[i-1].left == parent {
					path[i-1].left = rebuilt
				} else {
					path[i-1].right = rebuilt
				}
				break
			}
		}
	}
	return true, trace
}

// Delete removes key and reports whether it existed.
func (t *Tree[K, V]) Delete(key K) bool {
	var deleted bool
	t.root, deleted = t.delete(t.root, key)
	if !deleted {
		return false
	}
	t.n--
	if t.n == 0 {
		t.q = 0
		return true
	}
	if float64(t.n) < t.alpha*float64(t.q) {
		t.root = t.rebuild(t.root)
		t.q = t.n
	}
	return true
}

func (t *Tree[K, V]) delete(root *node[K, V], key K) (*node[K, V], bool) {
	if root == nil {
		return nil, false
	}
	if t.less(key, root.key) {
		var deleted bool
		root.left, deleted = t.delete(root.left, key)
		return root, deleted
	} else if t.less(root.key, key) {
		var deleted bool
		root.right, deleted = t.delete(root.right, key)
		return root, deleted
	}
	if root.left == nil {
		return root.right, true
	}
	if root.right == nil {
		return root.left, true
	}
	successor := root.right
	for successor.left != nil {
		successor = successor.left
	}
	root.key, root.value = successor.key, successor.value
	root.right, _ = t.delete(root.right, successor.key)
	return root, true
}

// InOrder returns all entries sorted by key.
func (t *Tree[K, V]) InOrder() []Entry[K, V] {
	entries := make([]Entry[K, V], 0, t.n)
	var visit func(*node[K, V])
	visit = func(current *node[K, V]) {
		if current == nil {
			return
		}
		visit(current.left)
		entries = append(entries, Entry[K, V]{Key: current.key, Value: current.value})
		visit(current.right)
	}
	visit(t.root)
	return entries
}

// Snapshot returns a copy of the tree shape for demos and visualizations.
func (t *Tree[K, V]) Snapshot() *NodeSnapshot[K, V] {
	return snapshot(t.root)
}

// Height returns -1 for an empty tree and 0 for a tree with only its root.
func (t *Tree[K, V]) Height() int { return height(t.root) }

// Stats returns counters useful for tests, benchmarks, and visualization.
func (t *Tree[K, V]) Stats() Stats {
	return Stats{t.n, t.q, t.Height(), t.rebuilds, t.rebuiltNodes}
}

func (t *Tree[K, V]) maxDepth(q int) int {
	if q <= 1 {
		return 0
	}
	return int(math.Floor(math.Log(float64(q)) / math.Log(1/t.alpha)))
}

func (t *Tree[K, V]) rebuild(root *node[K, V]) *node[K, V] {
	nodes := make([]*node[K, V], 0, size(root))
	flatten(root, &nodes)
	t.rebuilds++
	t.rebuiltNodes += len(nodes)
	return buildBalanced(nodes, 0, len(nodes))
}

func flatten[K, V any](root *node[K, V], nodes *[]*node[K, V]) {
	if root == nil {
		return
	}
	flatten(root.left, nodes)
	*nodes = append(*nodes, root)
	flatten(root.right, nodes)
}

func flattenKeys[K, V any](root *node[K, V], keys *[]K) {
	if root == nil {
		return
	}
	flattenKeys(root.left, keys)
	*keys = append(*keys, root.key)
	flattenKeys(root.right, keys)
}

func buildBalanced[K, V any](nodes []*node[K, V], start, end int) *node[K, V] {
	if start >= end {
		return nil
	}
	mid := start + (end-start)/2
	root := nodes[mid]
	root.left = buildBalanced(nodes, start, mid)
	root.right = buildBalanced(nodes, mid+1, end)
	return root
}

func snapshot[K, V any](root *node[K, V]) *NodeSnapshot[K, V] {
	if root == nil {
		return nil
	}
	return &NodeSnapshot[K, V]{
		Key:   root.key,
		Value: root.value,
		Left:  snapshot(root.left),
		Right: snapshot(root.right),
	}
}

func size[K, V any](root *node[K, V]) int {
	if root == nil {
		return 0
	}
	return 1 + size(root.left) + size(root.right)
}

func height[K, V any](root *node[K, V]) int {
	if root == nil {
		return -1
	}
	left, right := height(root.left), height(root.right)
	if left > right {
		return left + 1
	}
	return right + 1
}
