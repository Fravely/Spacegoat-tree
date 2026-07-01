// Package scapegoat implementa un árbol binario de búsqueda Scapegoat genérico.
package scapegoat

import (
	"cmp"
	"errors"
	"math"
)

// Ordered restringe claves que soportan comparación con <.
type Ordered = cmp.Ordered

// Entry es un par clave-valor devuelto por recorridos ordenados.
type Entry[K, V any] struct {
	Key   K
	Value V
}

// NodeSnapshot es una copia inmutable del árbol para serialización.
type NodeSnapshot[K, V any] struct {
	Key   K
	Value V
	Left  *NodeSnapshot[K, V]
	Right *NodeSnapshot[K, V]
}

// Stats resume el estado del árbol.
type Stats struct {
	Size         int
	MaxSize      int
	Height       int
	Rebuilds     int
	RebuiltNodes int
}

// InsertTrace registra decisiones algorítmicas de una inserción.
type InsertTrace[K any] struct {
	Updated     bool
	Inserted    bool
	Depth       int
	MaxDepth    int
	Path        []K
	Scapegoat   *K
	RebuiltKeys []K
}

type node[K, V any] struct {
	key         K
	value       V
	left, right *node[K, V]
}

// Tree es un Scapegoat Tree parametrizado por tipo de clave y valor.
type Tree[K, V any] struct {
	root         *node[K, V]
	less         func(K, K) bool
	alpha        float64
	n, q         int
	rebuilds     int
	rebuiltNodes int
}

// New crea un árbol con comparador personalizado.
func New[K, V any](alpha float64, less func(K, K) bool) (*Tree[K, V], error) {
	if less == nil {
		return nil, errors.New("scapegoat: less no puede ser nil")
	}
	if alpha <= 0.5 || alpha >= 1.0 {
		return nil, errors.New("scapegoat: alpha debe estar en (0.5, 1)")
	}
	return &Tree[K, V]{less: less, alpha: alpha}, nil
}

// NewOrdered crea un árbol para tipos ordenados nativos.
func NewOrdered[K Ordered, V any](alpha float64) (*Tree[K, V], error) {
	return New[K, V](alpha, func(a, b K) bool { return a < b })
}

// Alpha devuelve el factor de balance configurado.
func (t *Tree[K, V]) Alpha() float64 { return t.alpha }

// Len devuelve el número de nodos.
func (t *Tree[K, V]) Len() int { return t.n }

// Clear vacía el árbol.
func (t *Tree[K, V]) Clear() {
	t.root = nil
	t.n = 0
	t.q = 0
}

// Stats devuelve métricas actuales del árbol.
func (t *Tree[K, V]) Stats() Stats {
	return Stats{
		Size:         t.n,
		MaxSize:      t.q,
		Height:       t.Height(),
		Rebuilds:     t.rebuilds,
		RebuiltNodes: t.rebuiltNodes,
	}
}

func (t *Tree[K, V]) equal(a, b K) bool {
	return !t.less(a, b) && !t.less(b, a)
}

func (t *Tree[K, V]) maxDepth(q int) int {
	if q <= 1 {
		return 0
	}
	return int(math.Log(float64(q)) / math.Log(1.0/t.alpha))
}

// Insert inserta o actualiza una clave. Retorna true si la clave era nueva.
func (t *Tree[K, V]) Insert(key K, value V) bool {
	inserted, _ := t.InsertWithTrace(key, value)
	return inserted
}

// InsertWithTrace inserta con traza algorítmica completa.
func (t *Tree[K, V]) InsertWithTrace(key K, value V) (bool, InsertTrace[K]) {
	trace := InsertTrace[K]{}

	if t.root == nil {
		t.root = &node[K, V]{key: key, value: value}
		t.n = 1
		t.q = 1
		trace.Inserted = true
		trace.Depth = 0
		trace.MaxDepth = t.maxDepth(t.q)
		trace.Path = []K{key}
		return true, trace
	}

	path := make([]*node[K, V], 0, 16)
	current := t.root
	for {
		path = append(path, current)
		if t.equal(key, current.key) {
			current.value = value
			trace.Updated = true
			trace.Path = nodeKeys(path)
			trace.Depth = len(path) - 1
			trace.MaxDepth = t.maxDepth(t.q)
			return false, trace
		}
		if t.less(key, current.key) {
			if current.left == nil {
				current.left = &node[K, V]{key: key, value: value}
				path = append(path, current.left)
				break
			}
			current = current.left
		} else {
			if current.right == nil {
				current.right = &node[K, V]{key: key, value: value}
				path = append(path, current.right)
				break
			}
			current = current.right
		}
	}

	t.n++
	if t.n > t.q {
		t.q = t.n
	}

	trace.Inserted = true
	trace.Path = nodeKeys(path)
	depth := len(path) - 1
	trace.Depth = depth
	trace.MaxDepth = t.maxDepth(t.q)

	if depth > t.maxDepth(t.q) {
		for i := len(path) - 2; i >= 0; i-- {
			parent := path[i]
			child := path[i+1]
			if float64(t.size(child)) > t.alpha*float64(t.size(parent)) {
				sgKey := parent.key
				trace.Scapegoat = &sgKey
				t.flattenKeys(parent, &trace.RebuiltKeys)
				rebuilt := t.rebuild(parent)
				if i == 0 {
					t.root = rebuilt
				} else {
					gp := path[i-1]
					if gp.left == parent {
						gp.left = rebuilt
					} else {
						gp.right = rebuilt
					}
				}
				break
			}
		}
	}

	return true, trace
}

// Search busca una clave y devuelve su valor.
func (t *Tree[K, V]) Search(key K) (V, bool) {
	var zero V
	current := t.root
	for current != nil {
		if t.equal(key, current.key) {
			return current.value, true
		}
		if t.less(key, current.key) {
			current = current.left
		} else {
			current = current.right
		}
	}
	return zero, false
}

// SearchPath busca una clave y devuelve el camino recorrido.
func (t *Tree[K, V]) SearchPath(key K) (V, bool, []K) {
	var zero V
	var path []K
	current := t.root
	for current != nil {
		path = append(path, current.key)
		if t.equal(key, current.key) {
			return current.value, true, path
		}
		if t.less(key, current.key) {
			current = current.left
		} else {
			current = current.right
		}
	}
	return zero, false, path
}

// Delete elimina una clave. Retorna true si existía.
func (t *Tree[K, V]) Delete(key K) bool {
	newRoot, deleted := t.deleteNode(t.root, key)
	if !deleted {
		return false
	}
	t.root = newRoot
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

func (t *Tree[K, V]) deleteNode(root *node[K, V], key K) (*node[K, V], bool) {
	if root == nil {
		return nil, false
	}
	if t.less(key, root.key) {
		left, deleted := t.deleteNode(root.left, key)
		root.left = left
		return root, deleted
	}
	if t.less(root.key, key) {
		right, deleted := t.deleteNode(root.right, key)
		root.right = right
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
	root.key = successor.key
	root.value = successor.value
	right, _ := t.deleteNode(root.right, successor.key)
	root.right = right
	return root, true
}

// Height devuelve la altura (-1 si está vacío).
func (t *Tree[K, V]) Height() int {
	return height(t.root)
}

func height[K, V any](n *node[K, V]) int {
	if n == nil {
		return -1
	}
	lh := height(n.left)
	rh := height(n.right)
	if lh > rh {
		return lh + 1
	}
	return rh + 1
}

// InOrder devuelve las entradas en orden ascendente.
func (t *Tree[K, V]) InOrder() []Entry[K, V] {
	out := make([]Entry[K, V], 0, t.n)
	t.inOrder(t.root, &out)
	return out
}

func (t *Tree[K, V]) inOrder(n *node[K, V], out *[]Entry[K, V]) {
	if n == nil {
		return
	}
	t.inOrder(n.left, out)
	*out = append(*out, Entry[K, V]{Key: n.key, Value: n.value})
	t.inOrder(n.right, out)
}

// Snapshot construye una copia inmutable del árbol.
func (t *Tree[K, V]) Snapshot() *NodeSnapshot[K, V] {
	return snapshot(t.root)
}

func snapshot[K, V any](n *node[K, V]) *NodeSnapshot[K, V] {
	if n == nil {
		return nil
	}
	return &NodeSnapshot[K, V]{
		Key:   n.key,
		Value: n.value,
		Left:  snapshot(n.left),
		Right: snapshot(n.right),
	}
}

func (t *Tree[K, V]) size(n *node[K, V]) int {
	if n == nil {
		return 0
	}
	return 1 + t.size(n.left) + t.size(n.right)
}

func (t *Tree[K, V]) rebuild(root *node[K, V]) *node[K, V] {
	nodes := make([]*node[K, V], 0, t.size(root))
	t.flatten(root, &nodes)
	t.rebuilds++
	t.rebuiltNodes += len(nodes)
	return buildBalanced(nodes, 0, len(nodes))
}

func (t *Tree[K, V]) flatten(n *node[K, V], out *[]*node[K, V]) {
	if n == nil {
		return
	}
	t.flatten(n.left, out)
	*out = append(*out, n)
	t.flatten(n.right, out)
}

func (t *Tree[K, V]) flattenKeys(n *node[K, V], out *[]K) {
	if n == nil {
		return
	}
	t.flattenKeys(n.left, out)
	*out = append(*out, n.key)
	t.flattenKeys(n.right, out)
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

func nodeKeys[K, V any](path []*node[K, V]) []K {
	keys := make([]K, len(path))
	for i, n := range path {
		keys[i] = n.key
	}
	return keys
}
