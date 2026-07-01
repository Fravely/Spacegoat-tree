# Scapegoat Tree — Visualizador con backend en Go

Servidor en Go que expone una API REST para insertar, buscar, eliminar y
rebalancear un Scapegoat Tree, más una página web (`scapegoat-tree.html`)
que lo visualiza. Incluye carga de datos opcional desde SQL Server, con
**fallback automático** a 500 números de prueba si la conexión a la base
de datos falla por cualquier motivo.

---

## 1. Requisitos

- **Go** 1.21 o superior instalado. Verifica con:
  ```bash
  go version
  ```
  Si no lo tienes, descárgalo de https://go.dev/dl/

- (Opcional) Acceso a un **SQL Server** si quieres cargar datos reales
  en vez de los 500 números de prueba.

---

## 2. Estructura del proyecto

```
scapegoat-tree/
├── main.go                    # Punto de entrada (embebe la UI web)
├── config.json                # Configuración de SQL Server y tabla
├── scapegoat-tree.html        # Interfaz visual (fuente; se embebe al compilar)
├── go.mod / go.sum
├── scapegoat/                 # Núcleo algorítmico (genérico, sin HTTP ni BD)
│   ├── tree.go
│   ├── unit_test.go
│   ├── extra_test.go
│   └── benchmark_test.go
└── internal/
    ├── config/                # Carga de config.json
    ├── database/              # Operaciones SQL Server
    └── server/                # API REST + adaptador IntTree
```

El paquete `scapegoat/` es independiente de la web y de la base de datos:
puedes ejecutar `go test ./scapegoat/...` de forma aislada.

---

## 3. Instalación de dependencias

Desde la carpeta del proyecto:

```bash
go mod tidy
```

Esto descargará el driver de SQL Server (`github.com/microsoft/go-mssqldb`).

> **Nota:** el driver antiguo `github.com/denisenkom/go-mssqldb` está
> descontinuado. Este proyecto usa el sucesor oficial mantenido por
> Microsoft.

---

## 4. Configurar la conexión a SQL Server (opcional)

La conexión se define en **`config.json`** en la raíz del proyecto (o
junto al ejecutable compilado). Si el archivo no existe o la conexión
falla, el servidor **no se detiene** y usa 500 números de prueba (1–500).

Ejemplo de `config.json`:

```json
{
  "sqlserver": {
    "host": "localhost",
    "port": "1433",
    "useWindowsAuth": false,
    "user": "sa",
    "password": "TuPassword",
    "database": "ScapegoatDemo"
  },
  "table": {
    "name": "Productos",
    "keyColumn": "id"
  }
}
```

| Campo | Descripción |
|-------|-------------|
| `sqlserver.host` | Host o IP del servidor |
| `sqlserver.port` | Puerto (1433 por defecto) |
| `sqlserver.useWindowsAuth` | `true` para autenticación integrada de Windows |
| `sqlserver.user` / `password` | Credenciales SQL (si `useWindowsAuth` es `false`) |
| `sqlserver.database` | Nombre de la base de datos |
| `table.name` | Tabla de la que se leen los IDs |
| `table.keyColumn` | Columna entera que actúa como clave del árbol |

### Comportamiento si la conexión falla

Si las credenciales son incorrectas, el servidor no responde, hay timeout,
la tabla no existe, o cualquier otro error: el programa imprime una
advertencia y usa la secuencia de 500 números de prueba. Esto aplica al
arranque y a cada llamada a `/init-load` o `/load-bench`.

---

## 5. Ejecutar el servidor

Desde la carpeta del proyecto:

```bash
go run .
```

Si todo está bien, verás algo como:

```
✅ Configuración cargada desde "config.json": tabla="Productos", ...
✅ Conectado a SQL Server correctamente
Scapegoat Tree  →  http://localhost:8080
```

El navegador se abre automáticamente en Windows. La página HTML va
**embebida en el ejecutable**, así que no depende del directorio desde
el que lo ejecutes.

Si no hay conexión a la base de datos, el servidor arranca igual con
datos de prueba (1–500).

---

## 6. Abrir la interfaz visual

Con el servidor corriendo:

```
http://localhost:8080
```

La página se conecta a la API en `http://localhost:8080` automáticamente.

---

## 7. Endpoints disponibles

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/tree` | Estado actual del árbol |
| GET | `/insert?key=N` | Inserta la clave N (solo memoria) |
| POST | `/insert` | Inserta con JSON (sincroniza con BD si está activa) |
| GET | `/search?key=N` | Busca la clave N y devuelve el camino |
| GET | `/delete?key=N` | Elimina la clave N |
| GET | `/clear` | Vacía el árbol |
| GET | `/init-load` | Prepara cola de IDs (BD o fallback) |
| GET | `/step-load` | Inserta el siguiente ID de la cola |
| GET | `/load-bench` | Carga todos los IDs de golpe (benchmark HTTP) |
| GET | `/db-status` | Indica si la BD está activa |
| POST | `/db-toggle` | Activa/desactiva sincronización con BD |
| GET | `/table-columns` | Metadatos de columnas de la tabla |

---

## 8. Pruebas unitarias y benchmarks

El núcleo algorítmico vive en `scapegoat/` con pruebas y benchmarks
independientes del servidor web.

```bash
# Pruebas unitarias (16 tests)
go test ./scapegoat/... -v

# Benchmarks escalonados (1K, 5K, 10K, 50K nodos)
go test -bench=Benchmark -benchmem ./scapegoat/

# Solo inserción o búsqueda
go test -bench=BenchmarkInsert -benchmem ./scapegoat/
go test -bench=BenchmarkSearch -benchmem ./scapegoat/
```

| Benchmark | Qué mide |
|-----------|----------|
| `BenchmarkInsert` | Inserción ordenada a 1K–50K nodos |
| `BenchmarkInsertRandom` | Inserción aleatoria a 1K–50K nodos |
| `BenchmarkSearch` | Búsqueda en árboles de 1K–100K nodos |
| `BenchmarkDelete` | Inserción + eliminación completa |
| `BenchmarkInOrder` | Recorrido in-order |
| `BenchmarkMixedWorkload` | Carga mixta sobre 10K nodos |

---

## 9. Compilar un ejecutable (opcional)

```bash
go build -o scapegoat-server.exe .
./scapegoat-server        # Linux/Mac
scapegoat-server.exe      # Windows
```

Coloca `config.json` junto al `.exe` si quieres personalizar la conexión
a SQL Server fuera de la carpeta del proyecto.

---

## 10. Problemas comunes

- **`go: command not found`** → Go no está instalado o no está en el
  PATH. Reinstala desde https://go.dev/dl/ y reinicia la terminal.
- **El servidor arranca pero siempre usa los 500 números** → revisa
  `config.json` (host, credenciales, nombre de tabla/columna). Mira el
  mensaje de error en consola.
- **Puerto 8080 ocupado** → cierra la instancia anterior:
  ```powershell
  Get-NetTCPConnection -LocalPort 8080 | ForEach-Object { Stop-Process -Id $_.OwningProcess -Force }
  ```
  Luego vuelve a ejecutar `go run .`.
- **La página no carga** → confirma que el servidor esté corriendo y
  abre exactamente `http://localhost:8080` (con los dos puntos).
- **Error de firewall/puerto SQL** → confirma que el puerto 1433 esté
  accesible desde donde corres el programa.
