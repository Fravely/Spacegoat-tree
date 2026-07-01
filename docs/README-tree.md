# README — tree.go

## Qué es este archivo

Contiene la **lógica pura del Scapegoat Tree** (árbol de chivo
expiatorio). Es completamente independiente de SQL Server, HTTP o
JSON — solo trabaja con números enteros en memoria. Si algún día
quisieras usar este árbol en otro proyecto sin servidor web ni base de
datos, podrías copiar solo este archivo.

**No necesitas editar este archivo para uso normal.** No tiene nada
configurable como tabla o credenciales — es algoritmo puro.

---

## ¿Qué es un Scapegoat Tree, en una frase?

Es un árbol binario de búsqueda que se reordena solo cuando se
desbalancea demasiado, en vez de reordenarse en cada inserción (como
hacen los árboles AVL o Rojo-Negro). Cuando detecta que una rama se
volvió muy profunda, reconstruye esa rama entera de forma
perfectamente balanceada, usando como referencia un nodo "culpable":
el **chivo expiatorio** (de ahí el nombre).

---

## Estructuras de datos (líneas 1–20)

```go
type Node struct {
	Key    int    // el valor guardado (el ID)
	Left   *Node  // hijo izquierdo (valores menores)
	Right  *Node  // hijo derecho (valores mayores)
	Parent *Node  // puntero al padre — permite subir por el árbol sin recorrerlo desde la raíz
}

type ScapegoatTree struct {
	root    *Node    // la raíz del árbol
	size    int      // cuántos nodos hay AHORA
	maxSize int      // el tamaño más grande que tuvo el árbol desde el último rebuild total
	alpha   float64  // el factor de balance, fijo en 2/3 (0.666...)
}
```

`maxSize` es clave para el algoritmo de eliminación: el árbol no se
rebalancea en cada borrado, solo cuando el tamaño actual cae demasiado
respecto al tamaño máximo histórico — así evita rebalanceos
innecesarios.

---

## `Insert(key)` — Insertar

**Qué hace, paso a paso:**

1. Si el árbol está vacío, el nuevo nodo se vuelve la raíz directamente.
2. Si no, llama a `insertNode()`, que baja recursivamente por el árbol
   comparando valores (menor va a la izquierda, mayor a la derecha)
   hasta encontrar un lugar vacío. Devuelve la **profundidad absoluta**
   donde quedó el nuevo nodo (raíz = profundidad 0).
3. Calcula la altura máxima permitida con la fórmula del algoritmo:
   ```
   maxAllowed = log(tamaño) / log(1 / alpha)
   ```
   Con `alpha = 2/3`, esto es `log_{1.5}(n)` — la altura ideal de un
   árbol balanceado con ese factor.
4. **Si la profundidad real superó ese máximo**, el árbol está
   demasiado desbalanceado: busca el "chivo expiatorio" con
   `findScapegoat()` y reconstruye ese subárbol entero balanceado con
   `rebuild()`.
5. Devuelve tres valores: si se insertó, si hubo rebalanceo, y la
   clave del nodo que fue el chivo expiatorio (para que la página lo
   resalte visualmente).

**Duplicados:** si la clave ya existe, `insertNode` devuelve `-1` y
`Insert` responde `inserted = false` sin tocar nada.

---

## `findScapegoat(n)` — Encontrar al culpable

Sube desde el nodo recién insertado hacia la raíz, padre por padre. En
cada paso, compara el tamaño del subárbol del lado por el que venimos
contra el tamaño total del subárbol del padre. Si ese hijo representa
más del `alpha` (66.6%) del tamaño del padre, ese padre es el chivo
expiatorio — significa que esa rama creció demasiado desproporcionada.

Si sube hasta la raíz sin encontrar ningún desbalance así (caso raro),
devuelve `nil`, y quien llama (`Insert`) usa la raíz completa como
chivo expiatorio de respaldo.

---

## `rebuild(n)` — Reconstrucción balanceada

Esta es la operación que hace que el árbol vuelva a estar balanceado:

1. `flattenInOrder(n)` recorre el subárbol del chivo expiatorio
   **in-order** (izquierda, nodo, derecha), lo cual produce la lista
   de nodos **ya ordenada** de menor a mayor.
2. `buildBalanced(nodes, lo, hi)` toma esa lista ordenada y construye
   un árbol nuevo tomando siempre el elemento del medio como raíz de
   cada subárbol — esto garantiza la altura mínima posible.
3. El nuevo subárbol balanceado se reconecta al resto del árbol
   original en el lugar donde estaba el chivo expiatorio.
4. `fixParents()` recorre el subárbol nuevo arreglando todos los
   punteros `Parent`, que se habían perdido al reconstruir.

---

## `Search(key)` — Buscar

Recorrido clásico de árbol binario: compara, va a la izquierda o
derecha según corresponda, y registra cada nodo visitado en `path`
(usado por la página para animar el recorrido paso a paso). Devuelve
`found = true/false` y la ruta completa recorrida.

---

## `Delete(key)` — Eliminar

1. Encuentra el nodo con `findNode()`.
2. Lo elimina con `deleteNode()`, que maneja los tres casos clásicos de
   un árbol binario:
   - Nodo sin hijos: simplemente se desconecta del padre.
   - Nodo con un solo hijo: el hijo toma el lugar del nodo eliminado.
   - Nodo con dos hijos: se reemplaza por su **sucesor in-order** (el
     menor valor del subárbol derecho), y luego se elimina ese
     sucesor de su posición original.
3. **Rebalanceo perezoso:** después de borrar, si el tamaño actual cayó
   por debajo de `alpha * maxSize`, se reconstruye **todo el árbol**
   desde la raíz (no solo una rama, a diferencia de `Insert`). Esto
   evita que el árbol quede "hueco" después de muchos borrados.

---

## `Serialize()` — Convertir el árbol a JSON

Recorre el árbol y produce una lista plana de `SerializedNode`, cada
uno con su clave, la clave de su padre, y si es hijo izquierdo o
derecho. Esta es la estructura que la página web usa para dibujar el
árbol visualmente (no manda punteros de Go, que no tienen sentido en
JavaScript — manda solo las relaciones por clave).

---

## `Height()` / `Size()` / `Alpha()` — Consultas simples

Funciones de solo lectura usadas para mostrar las estadísticas en la
parte superior de la página (nodos, altura, alfa, rebalanceos).

---

## ¿Es correcto este algoritmo?

Sí. Cubre correctamente: cálculo de profundidad real vs. altura máxima
teórica permitida, búsqueda del chivo expiatorio comparando tamaños de
subárbol contra el padre directo (no acumulando de iteraciones
anteriores, que es un error común en otras implementaciones),
reconstrucción balanceada vía aplanado in-order + construcción por
punto medio, y rebalanceo perezoso en eliminación basado en el tamaño
máximo histórico. No requiere ningún cambio.

---

## Errores específicos de este archivo

Este archivo no se comunica con el exterior (ni HTTP ni SQL), así que
no genera errores de conexión ni de configuración. Si algo falla aquí,
sería un bug de lógica, no algo que el usuario final deba "arreglar"
editando algo — repórtalo si lo notas, pero no es un punto de
configuración.
