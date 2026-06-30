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

## 2. Estructura de archivos

```
scapegoat-tree/
├── main.go               # Servidor HTTP y endpoints
├── tree.go                # Lógica del Scapegoat Tree
├── scapegoat-tree.html    # Interfaz visual
├── go.mod                 # Dependencias del proyecto
└── README.md
```

Todos estos archivos deben estar en la **misma carpeta**.

---

## 3. Instalación de dependencias

Desde la carpeta del proyecto, ejecuta:

```bash
go mod tidy
```

Esto descargará automáticamente la dependencia del driver de SQL Server
(`github.com/microsoft/go-mssqldb`) y dejará el `go.mod`/`go.sum`
correctos.

Si prefieres instalarla manualmente:

```bash
go get github.com/microsoft/go-mssqldb
```

> **Nota:** el driver antiguo `github.com/denisenkom/go-mssqldb` está
> descontinuado. Este proyecto usa el sucesor oficial mantenido por
> Microsoft, que es compatible con el mismo API.

---

## 4. Configurar la conexión a SQL Server (opcional)

La conexión se configura con **variables de entorno**. Si no las
defines, el programa usa valores de relleno y, si la conexión falla,
**cae automáticamente** a generar 500 números de prueba (1 a 500) sin
detener el servidor.

| Variable             | Ejemplo           | Descripción                  |
|-----------------------|-------------------|-------------------------------|
| `SQLSERVER_HOST`      | `localhost`       | Host o IP del servidor        |
| `SQLSERVER_PORT`      | `1433`            | Puerto (1433 por defecto)     |
| `SQLSERVER_USER`      | `sa`              | Usuario de SQL Server         |
| `SQLSERVER_PASSWORD`  | `MiPassword123`   | Contraseña                    |
| `SQLSERVER_DATABASE`  | `MiBaseDeDatos`   | Nombre de la base de datos    |

### Windows (PowerShell)

```powershell
$env:SQLSERVER_HOST="localhost"
$env:SQLSERVER_PORT="1433"
$env:SQLSERVER_USER="sa"
$env:SQLSERVER_PASSWORD="TuPassword"
$env:SQLSERVER_DATABASE="TuBaseDeDatos"
```

### Linux / macOS

```bash
export SQLSERVER_HOST=localhost
export SQLSERVER_PORT=1433
export SQLSERVER_USER=sa
export SQLSERVER_PASSWORD=TuPassword
export SQLSERVER_DATABASE=TuBaseDeDatos
```

Estas variables deben configurarse **antes** de ejecutar `go run` o el
ejecutable compilado, en la misma terminal.

### Ajustar la consulta SQL

En `main.go`, dentro de `fetchIDsFromDB()`, ajusta esta línea a tu
tabla y columna reales:

```go
rows, err := db.QueryContext(ctx, "SELECT ID FROM TuTabla ORDER BY ID")
```

Por ejemplo, si tu tabla se llama `Clientes` y la columna `IdCliente`:

```go
rows, err := db.QueryContext(ctx, "SELECT IdCliente FROM Clientes ORDER BY IdCliente")
```

### Comportamiento si la conexión falla

Si las credenciales son incorrectas, el servidor no responde, hay un
timeout, la tabla no existe, o cualquier otro error: el programa
**no se detiene**. Imprime una advertencia en la consola y usa
automáticamente la secuencia de 500 números de prueba (1 al 500), tal
como hacía antes. Esto aplica tanto al iniciar el programa (intento de
conexión) como a cada llamada a `/init-load` o `/load-bench`.

---

## 5. Ejecutar el servidor

Desde la carpeta del proyecto:

```bash
go run main.go tree.go
```

Si todo está bien, verás algo como:

```
✅ Conectado a SQL Server correctamente
╔═══════════════════════════════════════════╗
║  Scapegoat Tree  →  http://localhost:8080 ║
╚═══════════════════════════════════════════╝
```

O, si no hay conexión a la base de datos:

```
⚠️  No se pudo conectar a SQL Server: ...
⚠️  El servidor seguirá funcionando con datos de prueba (1-500)
╔═══════════════════════════════════════════╗
║  Scapegoat Tree  →  http://localhost:8080 ║
╚═══════════════════════════════════════════╝
```

En ambos casos el servidor queda funcionando en `http://localhost:8080`.

---

## 6. Abrir la interfaz visual

Con el servidor corriendo, abre en tu navegador:

```
http://localhost:8080
```

Esto carga `scapegoat-tree.html`, que se conecta a la API en
`http://localhost:8080` automáticamente.

---

## 7. Endpoints disponibles

| Método | Endpoint       | Descripción                                          |
|--------|----------------|-------------------------------------------------------|
| GET    | `/tree`        | Devuelve el estado actual del árbol                   |
| GET    | `/insert?key=N`| Inserta la clave N                                    |
| GET    | `/search?key=N`| Busca la clave N y devuelve el camino recorrido       |
| GET    | `/delete?key=N`| Elimina la clave N                                    |
| GET    | `/clear`       | Vacía el árbol                                        |
| GET    | `/init-load`   | Prepara una cola de IDs (BD o fallback) para cargar paso a paso |
| GET    | `/step-load`   | Inserta el siguiente ID de la cola preparada           |
| GET    | `/load-bench`  | Carga todos los IDs de golpe (benchmark)               |

---

## 8. Compilar un ejecutable (opcional)

Si quieres un binario en vez de usar `go run` cada vez:

```bash
go build -o scapegoat-server main.go tree.go
./scapegoat-server        # Linux/Mac
scapegoat-server.exe      # Windows
```

---

## 9. Problemas comunes

- **`go: command not found`** → Go no está instalado o no está en el
  PATH. Reinstala desde https://go.dev/dl/ y reinicia la terminal.
- **El servidor arranca pero siempre usa los 500 números** → revisa que
  las variables de entorno estén bien escritas y definidas en la misma
  terminal donde ejecutas `go run`. Revisa también el mensaje de error
  que se imprime en consola; indica la causa exacta (credenciales,
  timeout, tabla inexistente, etc.).
- **Error de firewall/puerto** → confirma que el puerto de SQL Server
  (1433 por defecto) esté abierto y accesible desde donde corres el
  programa.
- **Puerto 8080 ocupado** → cierra el proceso que lo esté usando o
  cambia el puerto en la última línea de `main.go`
  (`http.ListenAndServe(":8080", nil)`).
