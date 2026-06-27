# Scapegoat Tree en Go
# hola
Implementacion desde cero del arbol **Scapegoat Tree** de Galperin y Rivest
(1993), realizada para el proyecto de Algoritmos y Estructuras de Datos.

## Que incluye

- Paquete generico `scapegoat` con claves y valores parametrizados.
- Insercion, busqueda, actualizacion y eliminacion.
- Reconstruccion balanceada de subarboles sin rotaciones.
- Nodos simples con clave, valor, hijo izquierdo e hijo derecho.
- Recorrido in-order, altura y estadisticas de reconstruccion.
- Pruebas unitarias, prueba aleatoria contra `map` y benchmarks.
- Demo que permite observar cuando ocurren reconstrucciones.

## Ejecutar

Se requiere Go 1.23 o posterior.

```bash
go test ./...
go test -bench=. ./scapegoat
go run ./cmd/demo
```

## Demo con SQL Server

La base de datos guarda los productos de forma persistente. El programa en Go
lee esos registros y construye un **Scapegoat Tree** en memoria usando `id`
como clave. Asi se demuestra la relacion entre base de datos e indice:

- SQL Server: almacenamiento permanente.
- Scapegoat Tree: busqueda ordenada y rapida en memoria.

Script opcional para crear la base y la tabla:

```sql
scripts/sqlserver/schema.sql
```

Variables de entorno usadas por la demo:

```powershell
$env:SQLSERVER_HOST="localhost"
$env:SQLSERVER_PORT=""
$env:SQLSERVER_INSTANCE=""
$env:SQLSERVER_DATABASE="InventarioProductosDB"
$env:SQLSERVER_USER="sa"
$env:SQLSERVER_PASSWORD="TuPassword"
$env:SQLSERVER_ENCRYPT="disable"
$env:SQLSERVER_TRUST_CERT="true"
```

Ejecutar la demo:

```bash
go run ./cmd/sqlserver-demo
```

El comando crea la base `InventarioProductosDB` si no existe, crea la tabla
`dbo.Productos` si no existe, inserta un dataset de ejemplo sin duplicar
registros, carga los productos al arbol y realiza una busqueda por `id`.

Si se necesita controlar exactamente la cadena de conexion, se puede usar
`SQLSERVER_DSN`:

```powershell
$env:SQLSERVER_DSN="server=localhost;user id=sa;password=TuPassword;database=InventarioProductosDB;encrypt=disable;TrustServerCertificate=true"
go run ./cmd/sqlserver-demo
```

## Simulacion Go + Vue

La simulacion reutiliza el paquete `scapegoat` desde un backend Go y muestra el
arbol en una pagina Vue. No requiere crear un proyecto con npm; Go sirve los
archivos estaticos desde la carpeta `web`.

```bash
go run ./cmd/simulation
```

Luego abrir:

```text
http://localhost:8080
```

La interfaz permite:

- insertar o actualizar productos usando el `id` como clave;
- buscar productos por `id`;
- eliminar productos;
- reiniciar el dataset de ejemplo;
- observar altura, cantidad de nodos, `q` y reconstrucciones.

Por defecto la simulacion usa memoria local. Si se define `SQLSERVER_DSN`,
la pagina trabaja contra SQL Server: al insertar o eliminar, primero actualiza
la tabla `dbo.Productos` y luego actualiza el Scapegoat Tree en memoria.

```powershell
$env:SQLSERVER_DSN="server=localhost;user id=sa;password=TuPassword;database=InventarioProductosDB;encrypt=disable;TrustServerCertificate=true"
go run ./cmd/simulation
```

## Idea del algoritmo

El arbol mantiene dos contadores:

- `n`: numero actual de nodos.
- `q`: maximo numero de nodos alcanzado desde la ultima reconstruccion global.

El parametro `alpha` cumple `0.5 < alpha < 1`. Al insertar, primero se realiza
una insercion de BST. Si la profundidad supera
`floor(log_(1/alpha)(q))`, se asciende por el camino hasta encontrar un nodo
desbalanceado: un hijo contiene mas de `alpha` veces los nodos de su padre.
Ese subarbol se aplana en orden y se reconstruye perfectamente balanceado.
Siguiendo la idea del articulo, el nodo no almacena un campo de tamano; cuando
el algoritmo necesita `size(u)`, lo calcula recorriendo el subarbol.

Al eliminar no se buscan chivos expiatorios. Cuando `n < alpha*q`, se
reconstruye todo el arbol y se asigna `q = n`.

## Complejidad

| Operacion | Peor caso individual | Amortizada |
|---|---:|---:|
| Busqueda | `O(log n)` | `O(log n)` |
| Insercion | `O(n)` por una reconstruccion | `O(log n)` |
| Eliminacion | `O(n)` por reconstruccion global | `O(log n)` |
| Espacio del arbol | `O(n)` | `O(n)` |
| Memoria auxiliar de reconstruccion | `O(k)` | para un subarbol de `k` nodos |

La ventaja frente a AVL o Red-Black es que no se almacena informacion de
balance por nodo y no se usan rotaciones. La contrapartida es que una operacion
individual puede reconstruir muchos nodos, aunque el costo por secuencia sea
amortizado.

## Estructura

```text
.
|-- cmd/demo/main.go
|-- cmd/simulation/main.go
|-- cmd/sqlserver-demo/main.go
|-- database/product.go
|-- database/sqlserver.go
|-- scapegoat/tree.go
|-- scapegoat/tree_test.go
|-- scapegoat/benchmark_test.go
|-- scripts/sqlserver/schema.sql
|-- web/index.html
|-- web/main.js
|-- web/styles.css
|-- go.mod
|-- go.sum
`-- README.md
```

## Uso desde otro paquete

```go
tree, err := scapegoat.NewOrdered[int, string](2.0 / 3.0)
if err != nil {
    log.Fatal(err)
}
tree.Insert(42, "dato")
value, found := tree.Search(42)
```

Para structs se puede proporcionar un comparador propio con `scapegoat.New`.

## Siguientes entregables

Este paquete es independiente de interfaz y persistencia. Por eso puede ser
reutilizado directamente por la aplicacion con base de datos y por el backend
Go de la simulacion en Vue.js, como exige el enunciado.
