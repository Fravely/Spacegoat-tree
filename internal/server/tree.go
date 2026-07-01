package server

import (
	"scapegoat-tree/scapegoat"
)

// IntTree adapta scapegoat.Tree[int, struct{}] a la API HTTP del proyecto.
type IntTree struct {
	inner *scapegoat.Tree[int, struct{}]
}

// NewIntTree crea un árbol de enteros con alpha = 2/3.
func NewIntTree() *IntTree {
	tree, err := scapegoat.NewOrdered[int, struct{}](2.0 / 3.0)
	if err != nil {
		panic(err)
	}
	return &IntTree{inner: tree}
}

// Insert inserta una clave y reporta si hubo rebalanceo.
func (t *IntTree) Insert(key int) (inserted bool, rebalanced bool, scapegoatKey int) {
	inserted, trace := t.inner.InsertWithTrace(key, struct{}{})
	if trace.Scapegoat != nil {
		rebalanced = true
		scapegoatKey = *trace.Scapegoat
	}
	return inserted, rebalanced, scapegoatKey
}

// Search busca una clave y devuelve el camino recorrido.
func (t *IntTree) Search(key int) (found bool, path []int) {
	_, found, path = t.inner.SearchPath(key)
	return found, path
}

// Delete elimina una clave y reporta si hubo rebalanceo global.
func (t *IntTree) Delete(key int) (deleted bool, rebalanced bool, scapegoatKey int) {
	statsBefore := t.inner.Stats()
	deleted = t.inner.Delete(key)
	if !deleted {
		return false, false, 0
	}
	statsAfter := t.inner.Stats()
	if statsAfter.Rebuilds > statsBefore.Rebuilds {
		rebalanced = true
		if snap := t.inner.Snapshot(); snap != nil {
			scapegoatKey = snap.Key
		}
	}
	return true, rebalanced, scapegoatKey
}

// Clear vacía el árbol.
func (t *IntTree) Clear() { t.inner.Clear() }

// Size devuelve el número de nodos.
func (t *IntTree) Size() int { return t.inner.Len() }

// Height devuelve la altura del árbol.
func (t *IntTree) Height() int { return t.inner.Height() }

// Alpha devuelve el factor de balance.
func (t *IntTree) Alpha() float64 { return t.inner.Alpha() }

// SerializedNode representa un nodo para la UI web.
type SerializedNode struct {
	Key       int    `json:"key"`
	ParentKey int    `json:"parentKey"`
	IsLeft    bool   `json:"isLeft"`
	Highlight string `json:"highlight"`
}

// Serialize aplana el árbol para JSON de la interfaz web.
func (t *IntTree) Serialize() []SerializedNode {
	snap := t.inner.Snapshot()
	result := make([]SerializedNode, 0, t.inner.Len())
	var walk func(n *scapegoat.NodeSnapshot[int, struct{}], parentKey int, isLeft bool)
	walk = func(n *scapegoat.NodeSnapshot[int, struct{}], parentKey int, isLeft bool) {
		if n == nil {
			return
		}
		result = append(result, SerializedNode{
			Key:       n.Key,
			ParentKey: parentKey,
			IsLeft:    isLeft,
		})
		walk(n.Left, n.Key, true)
		walk(n.Right, n.Key, false)
	}
	walk(snap, 0, false)
	return result
}
