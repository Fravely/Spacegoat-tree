# Scapegoat Tree + SQL Server — Documentación

Servidor en Go que mantiene un Scapegoat Tree en memoria, sincronizado
con una tabla de SQL Server. Permite buscar, insertar y eliminar nodos
desde una página web, reflejando esos cambios en la base de datos real.
Si la base de datos no está disponible, el sistema **no se detiene**:
sigue funcionando solo en memoria, con datos de prueba.

---

## 1. Qué hace cada archivo

| Archivo               | Qué hace |
|------------------------|----------|
| `tree.go`              | Lógica pura del Scapegoat Tree: insertar, buscar, eliminar, rebalancear. Solo trabaja con números enteros (IDs), no sabe nada de SQL Server. |
| `config.go`            | Lee `config.json` al arrancar. Si no existe o tiene errores, usa valores de relleno y avisa en consola, sin detener el programa. |
| `main.go`              | Servidor HTTP. Conecta a SQL Server, expone los endpoints (`/insert`, `/search`, `/delete`, etc.), y es quien decide si una operación también debe afectar la base de datos real. |
| `config.json`          | **Archivo editable.** Aquí defines tabla, columna ID, y credenciales de SQL Server. Lo editas tú directamente, sin tocar código Go. |
| `scapegoat-tree.html`  | Interfaz visual que consume la API y dibuja el árbol. |
| `go.mod` / `go.sum`    | Dependencias del proyecto (generadas por Go, no se editan a mano). |

---

## 2. Cómo funciona el flujo (lo que pediste)

### Al arrancar el servidor
1. Se lee `config.json` (tabla, columna ID, credenciales).
2. Se intenta conectar a SQL Server.
   - Si conecta: se cargan los IDs reales desde la tabla configurada.
   - Si falla (credenciales, servidor apagado, lo que sea): se usan 500
     números de prueba (1 a 500), y el servidor sigue funcionando
     normalmente — solo que sin sincronizar con la BD.

### Búsqueda (`/search?key=N`)
1. Busca primero en el **árbol en memoria** (rápido).
2. Si lo encuentra, hace un `SELECT * FROM [tabla] WHERE [columna] = N`
   contra SQL Server para traer **todas las columnas** de esa fila
   (sin que tengas que listarlas en config.json — se adapta solo).
3. Devuelve el resultado a la página, que lo muestra.

### Inserción (`/insert?key=N`)
1. Inserta N en el árbol en memoria.
2. Si hay conexión, también hace `INSERT INTO [tabla] ([columna]) VALUES (N)`
   en la base real. Las demás columnas quedan `NULL` porque la página
   solo provee el ID — esto es esperado.
3. Si el INSERT en la base de datos falla (ej: llave duplicada), se
   **revierte** la inserción en memoria para que ambos queden
   consistentes, y se informa el error.

### Eliminación (`/delete?key=N`)
1. Si hay conexión, primero hace `DELETE FROM [tabla] WHERE [columna] = N`
   en la base real.
2. Si eso funciona, entonces elimina el nodo del árbol en memoria.
3. Si el DELETE en la base de datos falla, el árbol **no se toca**, para
   no desincronizar memoria y base de datos.

### Sin conexión a la base de datos
Todas las operaciones (insertar, buscar, eliminar) siguen funcionando
**solo en memoria**. La respuesta incluye un aviso (`dbWarning`) para
que sepas que esa operación no llegó a la base de datos real.

---

## 3. Cómo editar la configuración (`config.json`)

Este es el único archivo que necesitas tocar para cambiar de tabla,
columna o credenciales — no requiere recompilar nada, solo reiniciar
el servidor.

```json
{
  "sqlserver": {
    "host": "localhost",
    "port": "1433",
    "user": "sa",
    "password": "changeme",
    "database": "MiBaseDeDatos"
  },
  "table": {
    "name": "TuTabla",
    "keyColumn": "ID"
  }
}
```

| Campo                    | Qué es |
|----------------------------|--------|
| `sqlserver.host`          | Dirección del servidor SQL (ej: `localhost`, o una IP) |
| `sqlserver.port`          | Puerto, normalmente `1433` |
| `sqlserver.user`          | Usuario de SQL Server |
| `sqlserver.password`      | Contraseña |
| `sqlserver.database`      | Nombre de la base de datos donde está tu tabla |
| `table.name`              | Nombre exacto de la tabla a usar |
| `table.keyColumn`         | Nombre de la columna llave primaria (debe ser un número entero) |

**Para cambiar de tabla rápidamente:** edita `table.name` y
`table.keyColumn`, guarda el archivo, y reinicia el servidor
(`Ctrl+C` y vuelve a correr `go run`). No hay que tocar ningún `.go`.

Si borras o dañas `config.json`, el programa no se cae: usa valores de
relleno (tabla `TuTabla`, columna `ID`, host `localhost`) y te avisa en
consola que debes crear el archivo.

---

## 4. Requisitos e instalación

- **Go 1.21+** instalado (`go version` para verificar).
- Acceso a un servidor SQL Server (opcional — si no lo tienes, el
  sistema usa datos de prueba automáticamente).

Desde la carpeta donde están todos los archivos:

```bash
go mod tidy
```

Esto descarga automáticamente todas las dependencias necesarias
(`github.com/microsoft/go-mssqldb` y sus dependencias internas, como el
SDK de Azure para autenticación). Es normal ver varias líneas de
`go: downloading ...` — Go las maneja solo, no hace falta instalar nada
manualmente aparte de eso.

---

## 5. Ejecutar el servidor

```bash
go run main.go tree.go config.go
```

Verás algo como:

```
✅ Configuración cargada: tabla="Clientes", columna="IdCliente", host="localhost", database="MiBD"
✅ Conectado a SQL Server correctamente
╔═══════════════════════════════════════════╗
║  Scapegoat Tree  →  http://localhost:8080 ║
╚═══════════════════════════════════════════╝
```

O, si no hay conexión disponible:

```
⚠️  No se pudo conectar a SQL Server: ...
⚠️  El servidor seguirá funcionando con datos de prueba (1-500) y sin sincronizar con la BD
╔═══════════════════════════════════════════╗
║  Scapegoat Tree  →  http://localhost:8080 ║
╚═══════════════════════════════════════════╝
```

Luego abre en el navegador: **http://localhost:8080**

---

## 6. Compilar un ejecutable (opcional)

```bash
go build -o scapegoat-server main.go tree.go config.go
./scapegoat-server        # Linux/Mac
scapegoat-server.exe      # Windows
```

`config.json` debe quedar en la misma carpeta que el ejecutable.

---

## 7. Errores comunes y solución

### `go: go.mod file not found`
Estás ejecutando el comando en una carpeta donde no está `go.mod`. Ve a
la carpeta correcta con `cd` o ejecuta `go mod init scapegoat-tree`
seguido de `go mod tidy` en esa carpeta.

### El programa siempre usa los 500 números de prueba
Revisa el mensaje de advertencia que se imprime en consola al arrancar
— indica la causa exacta. Las causas más comunes:
- `config.json` no está en la misma carpeta que el ejecutable.
- Usuario/contraseña incorrectos en `config.json`.
- El servidor SQL Server no está corriendo o el puerto está bloqueado
  por un firewall.
- El nombre de tabla o columna en `config.json` no existe exactamente
  así en la base de datos (SQL Server es sensible a mayúsculas según
  la configuración del servidor).

### `no se pudo insertar en la base de datos: ... PRIMARY KEY` (o similar)
Quiere decir que el ID que intentas insertar ya existe como fila real
en la tabla, aunque el árbol en memoria no lo tuviera. El sistema
revierte la inserción en memoria automáticamente para mantener todo
sincronizado. Verifica el ID antes de insertarlo de nuevo.

### `no se pudo eliminar de la base de datos`
Puede pasar si la fila ya no existe en la BD (fue borrada por otro
medio), o si hay una restricción de llave foránea en SQL Server que
impide eliminarla mientras otra tabla la referencia. El árbol en
memoria no se modifica en este caso, para evitar desincronización.

### El puerto 8080 ya está en uso
Cierra el proceso que lo esté usando, o cambia el puerto editando la
última línea de `main.go`: `http.ListenAndServe(":8080", nil)` por el
puerto que prefieras (ej: `:9090`).

### Error de compilación al correr `go run`
Asegúrate de incluir los tres archivos `.go` en el comando:
```bash
go run main.go tree.go config.go
```
Si falta alguno, Go no encontrará las funciones que usa.

### `go: downloading ...` se queda colgado o falla
Necesitas conexión a internet la primera vez que corres `go mod tidy`
(para descargar las dependencias). Las ejecuciones posteriores no
necesitan internet, ya quedan en caché local.

### Las columnas no se ven en la búsqueda
Confirma que haya conexión exitosa a SQL Server (revisa el mensaje al
arrancar el servidor). Sin conexión, el sistema avisa con un mensaje
("Sin conexión a BD...") en vez de mostrar columnas, porque no hay
forma de consultarlas.
