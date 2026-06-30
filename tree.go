package main

import (
	"fmt"
	"math"
)

type Node struct {
	Key    int
	Left   *Node
	Right  *Node
	Parent *Node
}

type ScapegoatTree struct {
	root    *Node
	size    int
	maxSize int
	alpha   float64
}

func NewScapegoatTree() *ScapegoatTree {
	return &ScapegoatTree{alpha: 2.0 / 3.0}
}

//  INSERTAR

func (t *ScapegoatTree) Insert(key int) (inserted bool, rebalanced bool, scapegoatKey int) {
	newNode := &Node{Key: key}

	if t.root == nil {
		t.root = newNode
		t.size = 1
		t.maxSize = 1
		return true, false, 0
	}

	// depth = profundidad ABSOLUTA desde la raíz (raíz = profundidad 0)
	depth := t.insertNode(t.root, newNode, 0)
	if depth == -1 {
		return false, false, 0
	}

	t.size++
	if t.size > t.maxSize {
		t.maxSize = t.size
	}

	// Altura máxima permitida por el algoritmo: log_{1/α}(n)
	// Con α=2/3: log_{1.5}(n)
	maxAllowed := math.Log(float64(t.size)) / math.Log(1.0/t.alpha)
	fmt.Printf("📊 Insert %d | depth=%d | maxAllowed=%.2f | size=%d\n",
		key, depth, maxAllowed, t.size)

	if float64(depth) > maxAllowed {
		scapegoat := t.findScapegoat(newNode)
		if scapegoat == nil {
			// Fallback: el árbol entero está desbalanceado → reconstruir desde raíz
			scapegoat = t.root
		}
		scapegoatKey = scapegoat.Key
		fmt.Printf("🔴 Chivo expiatorio: nodo %d | subárbol=%d nodos\n",
			scapegoat.Key, t.subtreeSize(scapegoat))
		t.rebuild(scapegoat)
		rebalanced = true
	}

	return true, rebalanced, scapegoatKey
}

// insertNode inserta recursivamente.
// Devuelve la profundidad ABSOLUTA del nuevo nodo (raíz=0, hijo de raíz=1, etc.)
// Devuelve -1 si la clave es duplicada.
func (t *ScapegoatTree) insertNode(current, newNode *Node, depth int) int {
	if newNode.Key == current.Key {
		return -1
	}
	if newNode.Key < current.Key {
		if current.Left == nil {
			current.Left = newNode
			newNode.Parent = current
			return depth + 1
		}
		return t.insertNode(current.Left, newNode, depth+1)
	}
	if current.Right == nil {
		current.Right = newNode
		newNode.Parent = current
		return depth + 1
	}
	return t.insertNode(current.Right, newNode, depth+1)
}

//  BUSCAR

func (t *ScapegoatTree) Search(key int) (found bool, path []int) {
	current := t.root
	for current != nil {
		path = append(path, current.Key)
		if key == current.Key {
			return true, path
		} else if key < current.Key {
			current = current.Left
		} else {
			current = current.Right
		}
	}
	return false, path
}

//  ELIMINAR

func (t *ScapegoatTree) Delete(key int) (deleted bool, rebalanced bool, scapegoatKey int) {
	node := t.findNode(t.root, key)
	if node == nil {
		return false, false, 0
	}
	t.deleteNode(node)
	t.size--

	// Rebalanceo perezoso: tamaño cayó por debajo de α·maxSize
	if t.size > 0 && float64(t.size) < t.alpha*float64(t.maxSize) {
		fmt.Printf("🔴 Delete rebalanceo | size=%d maxSize=%d\n", t.size, t.maxSize)
		t.rebuild(t.root)
		t.maxSize = t.size
		rebalanced = true
		if t.root != nil {
			scapegoatKey = t.root.Key
		}
	}
	return true, rebalanced, scapegoatKey
}

func (t *ScapegoatTree) findNode(n *Node, key int) *Node {
	if n == nil {
		return nil
	}
	if key == n.Key {
		return n
	} else if key < n.Key {
		return t.findNode(n.Left, key)
	}
	return t.findNode(n.Right, key)
}

func (t *ScapegoatTree) deleteNode(n *Node) {
	if n.Left == nil && n.Right == nil {
		t.replaceChild(n.Parent, n, nil)
		return
	}
	if n.Left == nil {
		n.Right.Parent = n.Parent
		t.replaceChild(n.Parent, n, n.Right)
		return
	}
	if n.Right == nil {
		n.Left.Parent = n.Parent
		t.replaceChild(n.Parent, n, n.Left)
		return
	}
	// Dos hijos: reemplazar con sucesor in-order
	successor := n.Right
	for successor.Left != nil {
		successor = successor.Left
	}
	n.Key = successor.Key
	t.deleteNode(successor)
}

func (t *ScapegoatTree) replaceChild(parent, old, newNode *Node) {
	if parent == nil {
		t.root = newNode
		if newNode != nil {
			newNode.Parent = nil
		}
		return
	}
	if parent.Left == old {
		parent.Left = newNode
	} else {
		parent.Right = newNode
	}
	if newNode != nil {
		newNode.Parent = parent
	}
}

//  REBALANCEO

// findScapegoat sube desde el nodo insertado buscando el ancestro desbalanceado.
// CORRECCIÓN CLAVE: comparamos el tamaño del hijo DIRECTO (left o right del padre),
// NO una variable acumulada de iteraciones anteriores.
func (t *ScapegoatTree) findScapegoat(n *Node) *Node {
	current := n

	for current.Parent != nil {
		parent := current.Parent

		leftSize := t.subtreeSize(parent.Left)
		rightSize := t.subtreeSize(parent.Right)
		parentSize := 1 + leftSize + rightSize

		// ¿Cuál hijo somos? Usar el tamaño real de ese subárbol
		var childSize int
		if parent.Left == current {
			childSize = leftSize
		} else {
			childSize = rightSize
		}

		// Condición scapegoat del algoritmo
		if float64(childSize) > t.alpha*float64(parentSize) {
			return parent
		}

		current = parent
	}

	// No se encontró ningún ancestro desbalanceado → el llamador usará t.root
	return nil
}

func (t *ScapegoatTree) subtreeSize(n *Node) int {
	if n == nil {
		return 0
	}
	return 1 + t.subtreeSize(n.Left) + t.subtreeSize(n.Right)
}

// rebuild reconstruye el subárbol de n en forma perfectamente balanceada
func (t *ScapegoatTree) rebuild(n *Node) {
	flat := t.flattenInOrder(n)
	if len(flat) == 0 {
		return
	}
	parentOfN := n.Parent
	newSub := t.buildBalanced(flat, 0, len(flat)-1)

	// Reconectar el nuevo subárbol al resto del árbol
	if parentOfN == nil {
		t.root = newSub
	} else if parentOfN.Left == n {
		parentOfN.Left = newSub
	} else {
		parentOfN.Right = newSub
	}

	// Fijar todos los punteros Parent del subárbol reconstruido
	t.fixParents(newSub, parentOfN)
}

func (t *ScapegoatTree) flattenInOrder(n *Node) []*Node {
	if n == nil {
		return nil
	}
	left := t.flattenInOrder(n.Left)
	right := t.flattenInOrder(n.Right)
	return append(append(left, n), right...)
}

func (t *ScapegoatTree) buildBalanced(nodes []*Node, lo, hi int) *Node {
	if lo > hi {
		return nil
	}
	mid := (lo + hi) / 2
	root := nodes[mid]
	root.Left = t.buildBalanced(nodes, lo, mid-1)
	root.Right = t.buildBalanced(nodes, mid+1, hi)
	return root
}

func (t *ScapegoatTree) fixParents(n *Node, parent *Node) {
	if n == nil {
		return
	}
	n.Parent = parent
	t.fixParents(n.Left, n)
	t.fixParents(n.Right, n)
}

//  LIMPIAR

func (t *ScapegoatTree) Clear() {
	t.root = nil
	t.size = 0
	t.maxSize = 0
}

//  SERIALIZAR

type SerializedNode struct {
	Key       int    `json:"key"`
	ParentKey int    `json:"parentKey"`
	IsLeft    bool   `json:"isLeft"` // ← NUEVO: indica si es hijo izquierdo
	Highlight string `json:"highlight"`
}

func (t *ScapegoatTree) Serialize() []SerializedNode {
	result := []SerializedNode{}
	t.serializeNode(t.root, 0, false, &result)
	return result
}

func (t *ScapegoatTree) serializeNode(n *Node, parentKey int, isLeft bool, result *[]SerializedNode) {
	if n == nil {
		return
	}
	*result = append(*result, SerializedNode{
		Key:       n.Key,
		ParentKey: parentKey,
		IsLeft:    isLeft,
	})
	t.serializeNode(n.Left, n.Key, true, result)
	t.serializeNode(n.Right, n.Key, false, result)
}

func (t *ScapegoatTree) Height() int {
	return t.heightNode(t.root)
}

func (t *ScapegoatTree) heightNode(n *Node) int {
	if n == nil {
		return 0
	}
	l := t.heightNode(n.Left)
	r := t.heightNode(n.Right)
	if l > r {
		return l + 1
	}
	return r + 1
}

func (t *ScapegoatTree) Size() int      { return t.size }
func (t *ScapegoatTree) Alpha() float64 { return t.alpha }
