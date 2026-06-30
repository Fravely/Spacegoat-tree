# README — main.go

## Qué es este archivo

Es el **servidor HTTP**. Es el corazón que conecta tres cosas: el
árbol en memoria (`tree.go`), la configuración (`config.go` +
`config.json`), y SQL Server. Cuando la página web (`scapegoat-tree.html`)
hace clic en un botón, termina llamando a una de las funciones de este
archivo.

No necesitas editar este archivo para uso normal — todo lo
personalizable (tabla, columna, credenciales) vive en `config.json`.
Solo lo tocarías si quisieras cambiar el comportamiento del programa
(por ejemplo, agregar un endpoint nuevo).

---

## Estructura general, de arriba a abajo

### 1. Imports (líneas 1–15)

```go
import (
	"context"        // para poner límites de tiempo (timeouts) a las consultas
	"database/sql"   // API estándar de Go para hablar con bases de datos
	"encoding/json"  // convertir datos de Go a JSON y viceversa
	"fmt"            // formatear texto y errores
	"log"            // imprimir mensajes en consola (avisos, errores)
	"net/http"       // crear el servidor web
	"strconv"        // convertir texto a número (ej: "20" -> 20)
	"sync"           // sync.Mutex, evita que dos peticiones choquen al mismo tiempo
	"time"           // duración de timeouts

	_ "github.com/microsoft/go-mssqldb" // el driver de SQL Server
)
```

El `_` antes de `github.com/microsoft/go-mssqldb` es una particularidad
de Go: significa "importa este paquete solo para que se registre a sí
mismo internamente" (el driver se registra ante `database/sql` al
cargarse), aunque el código nunca llame a sus funciones directamente
por nombre.

### 2. Estado global (líneas 17–27)

```go
var (
	tree      *ScapegoatTree  // el árbol en memoria (definido en tree.go)
	db        *sql.DB         // la conexión activa a SQL Server (nil si no hay conexión)
	mu        sync.Mutex      // candado para evitar que dos peticiones modifiquen el árbol a la vez
	loadQueue []int           // cola de IDs pendientes por insertar (usado por /init-load y /step-load)
	loadIndex int             // en qué posición de la cola vamos
	loadTotal int              // cuántos IDs hay en total en la cola
)
```

**¿Por qué un Mutex (`mu`)?** Go puede atender varias peticiones HTTP
al mismo tiempo (en hilos separados). Si dos personas insertan un nodo
exactamente al mismo tiempo sin protección, el árbol podría corromperse
(condición de carrera). Por eso cada handler hace `mu.Lock()` al
empezar y `defer mu.Unlock()` para liberar el candado automáticamente
al terminar, garantizando que las operaciones sobre el árbol ocurran
una por una.

### 3. Estructuras de respuesta (líneas 29–65)

`TreeResponse` define el formato JSON que se envía de vuelta a la
página después de cada operación (insertar, buscar, eliminar):

```go
type TreeResponse struct {
	TreeNodes      []SerializedNode `json:"treeNodes"`      // el árbol completo, listo para dibujar
	Size           int              `json:"size"`           // cuántos nodos tiene
	Height         int              `json:"height"`         // altura actual del árbol
	Alpha          float64          `json:"alpha"`          // el factor de balance (2/3 por defecto)
	Rebalanced     bool             `json:"rebalanced"`     // ¿esta operación causó un rebalanceo?
	RebalanceCount int              `json:"rebalanceCount"` // cuántos rebalanceos hubo (carga masiva)
	ScapegoatKey   int              `json:"scapegoatKey"`   // qué nodo fue el "chivo expiatorio"
	DBWarning      string           `json:"dbWarning,omitempty"` // aviso si la BD no participó en la operación
}
```

El `json:"..."` después de cada campo le dice a Go cómo se debe llamar
ese campo cuando se convierte a JSON (la página JavaScript espera
exactamente esos nombres). `omitempty` en `DBWarning` significa: si
está vacío, ni siquiera lo incluyas en el JSON final (para no ensuciar
la respuesta cuando no hay advertencia).

### 4. `cors()` (línea 67)

```go
func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	...
}
```

CORS es un mecanismo de seguridad de los navegadores. Esta función le
dice al navegador "está bien que la página web le hable a este
servidor desde cualquier origen". Se llama al inicio de cada handler.

---

## Los Handlers (funciones que atienden cada botón de la página)

Un "handler" es la función que se ejecuta cuando llega una petición a
una URL específica. Se registran al final del archivo, en `main()`.

### `handleTree` → `GET /tree`
Solo devuelve el estado actual del árbol, sin modificar nada. Se usa
para refrescar la vista.

### `handleInsert` → `GET /insert?key=N`
**Qué hace, paso a paso:**
1. Lee el número `N` de la URL (`strconv.Atoi`, que convierte texto a
   entero; si no es un número válido, responde error 400).
2. Bloquea el Mutex.
3. Inserta `N` en el árbol en memoria (`tree.Insert(key)`).
4. Si la clave ya existía, responde error 409 (conflicto) sin tocar la
   base de datos.
5. **Si hay conexión a la BD** (`db != nil`): intenta también
   `INSERT INTO [tabla] ([columna]) VALUES (N)` en SQL Server real.
   - Si ese INSERT falla (ej: la fila ya existe en la BD aunque no
     estuviera en el árbol), se **revierte** la inserción en memoria
     (`tree.Delete(key)`) para que ambos queden sincronizados, y se
     responde error 500 con el detalle.
6. **Si NO hay conexión**: la inserción queda solo en memoria, y se
   agrega un aviso `dbWarning` a la respuesta para que la página lo
   muestre.

### `handleSearch` → `GET /search?key=N`
**Qué hace, paso a paso:**
1. Busca `N` en el árbol en memoria (`tree.Search(key)`), que devuelve
   si lo encontró y el camino recorrido (para la animación visual).
2. **Si lo encontró Y hay conexión a la BD:** llama a
   `fetchRowFromDB(key)`, que hace `SELECT * FROM [tabla] WHERE [columna] = N`
   y trae **todas las columnas** de esa fila real (no solo el ID).
3. Esa fila completa se agrega a la respuesta bajo la clave `"row"`,
   que la página web usa para mostrar el modal con la tabla bonita.
4. Si no hay conexión, o si encontró el nodo pero ya no existe en la
   BD, se agrega un `dbWarning` explicando por qué no hay datos extra.

### `handleDelete` → `GET /delete?key=N`
**Orden importante:** primero intenta borrar en la base de datos real
(`DELETE FROM [tabla] WHERE [columna] = N`), y **solo si eso funciona**
(o si no hay conexión) borra el nodo del árbol en memoria. Esto evita
que el árbol diga que algo no existe mientras la fila real sigue en la
base de datos.

### `handleClear` → `GET /clear`
Solo llama a `tree.Clear()` (vacía el árbol en memoria) y reinicia la
cola de carga. **No toca la base de datos para nada.**

### `handleInitLoad` / `handleStepLoad` → carga paso a paso (demo visual)
`handleInitLoad` limpia el árbol, trae los IDs reales de la BD (vía
`fetchIDsFromDB()`) y los guarda en `loadQueue`. `handleStepLoad` va
insertando un ID de esa cola cada vez que se llama — así la página
puede animar la inserción nodo por nodo.

### `handleBenchLoad` → `GET /load-bench`
Igual que arriba, pero inserta **todos** los IDs de golpe, sin pausas
(usado para pruebas de rendimiento/benchmark).

---

## Las funciones de conexión a SQL Server

### `initDB()`
Arma el connection string (llamando a `buildConnString()`, que vive en
`config.go`), intenta abrir la conexión, y hace un `PingContext` con
límite de 5 segundos para confirmar que el servidor responde de
verdad (`sql.Open` no valida la conexión por sí solo, solo la prepara).
Si algo falla, devuelve el error — quien la llama (`main()`) decide
qué hacer.

### `fetchIDsFromDB()`
Trae los IDs de la tabla configurada con:
```sql
SELECT [columna] FROM [tabla] ORDER BY [columna]
```
**Comportamiento de fallback:** si `db` es `nil`, o si la consulta
falla por cualquier motivo (credenciales, tabla inexistente, timeout,
etc.), imprime una advertencia en consola y devuelve
`fallbackIDs()` (500 números del 1 al 500) — el programa nunca se cae
por esto.

### `fetchRowFromDB(key)`
Trae la fila completa de un ID específico con:
```sql
SELECT * FROM [tabla] WHERE [columna] = @p1
```
El `@p1` es un parámetro seguro (evita inyección SQL) — Go lo
reemplaza internamente por el valor de `key`. Usa `rows.Columns()`
para descubrir dinámicamente **qué columnas tiene la tabla**, sin que
tengas que listarlas en ningún lado: así funciona igual con una tabla
de 3 columnas que con una de 20. Convierte los resultados a un
`map[string]interface{}` (columna → valor) listo para mandar como
JSON.

### `insertRowInDB(key)` / `deleteRowFromDB(key)`
Ejecutan `INSERT` y `DELETE` reales sobre la tabla configurada, usando
siempre parámetros seguros (`@p1`), nunca concatenando el valor
directamente en el texto SQL (eso evita inyección SQL).

### `fallbackIDs()`
Genera el arreglo `[1, 2, 3, ..., 500]`. Es el "plan B" que se usa
cuando no hay base de datos disponible.

---

## `main()` — el punto de entrada

```go
func main() {
	loadConfig()              // 1. Lee config.json
	tree = NewScapegoatTree()  // 2. Crea el árbol vacío

	conn, err := initDB()      // 3. Intenta conectar a SQL Server
	if err != nil {
		db = nil  // sin conexión, modo solo-memoria
	} else {
		db = conn
		defer db.Close()  // cierra la conexión al terminar el programa
	}

	http.HandleFunc("/tree", handleTree)   // 4. Registra cada endpoint
	...

	http.Handle("/", http.FileServer(http.Dir("./")))  // 5. Sirve el HTML

	log.Fatal(http.ListenAndServe(":8080", nil))  // 6. Arranca el servidor
}
```

`http.FileServer(http.Dir("./"))` es lo que permite que al abrir
`http://localhost:8080` se cargue automáticamente
`scapegoat-tree.html` — sirve cualquier archivo estático de la carpeta
actual.

---

## Qué puedes configurar aquí (y qué NO)

**No edites este archivo para cambiar tabla, columna o credenciales**
— eso va en `config.json` (ver `README-config.md`).

Lo único que tendría sentido tocar manualmente en `main.go`:

| Qué cambiar | Dónde | Cómo |
|---|---|---|
| Puerto del servidor web (no el de SQL Server) | Última línea de `main()` | Cambia `":8080"` por ejemplo a `":9090"` |
| Tiempo de espera (timeout) de las consultas | `initDB()`, `fetchIDsFromDB()`, etc. | Cambia el número en `time.Second` |

---

## Errores específicos de este archivo

### `la clave ya existe en el árbol` (409)
Intentaste insertar un ID que el árbol en memoria ya tiene. No es un
error de la base de datos, es una validación normal del árbol.

### `no se pudo insertar en la base de datos: ...` (500)
El `INSERT` falló en SQL Server (ej: llave primaria duplicada en la
tabla real, aunque el árbol no la tuviera). El programa revierte
automáticamente la inserción en memoria para que ambos queden
sincronizados — no hace falta que tú hagas nada manual.

### `nodo no encontrado en el árbol` (404)
Intentaste eliminar un ID que no existe en el árbol en memoria.

### El servidor arranca pero `/` da 404 o no carga la página
Verifica que `scapegoat-tree.html` esté en la **misma carpeta** desde
donde ejecutas `go run` — `http.FileServer` sirve archivos relativos a
la carpeta actual, no a donde está `main.go` si los ejecutas desde
otro lado.

### El puerto 8080 ya está en uso
Cierra el proceso que lo esté usando o cambia `:8080` por otro puerto
libre en la última línea de `main()`.
