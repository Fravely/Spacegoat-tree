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

### Botón "Limpiar" (`/clear`)
Solo borra el árbol **en memoria**. No envía ningún `DELETE` a SQL
Server — tu base de datos queda intacta. Es solo para reiniciar la
visualización.

### Volver a cargar después de modificar datos
La carga masiva y la carga paso a paso (`/load-bench`, `/init-load`)
**siempre consultan la tabla real en ese momento**, no datos guardados
de una carga anterior. Por eso, si insertaste o eliminaste nodos y esas
operaciones llegaron a la base de datos (es decir, sin que apareciera
el aviso de "sin conexión"), la próxima carga reflejará la tabla tal
como está ahora, con esos cambios incluidos.



## 3. Cómo editar la configuración (`config.json`)

**Toda la configuración de conexión y autenticación se edita en un solo
lugar: `config.json`. Nunca hace falta tocar `config.go`, `main.go` ni
ningún otro archivo `.go` para cambiar credenciales, tabla, columna o
tipo de autenticación.**

Este es el único archivo que necesitas tocar para cambiar de tabla,
columna o credenciales — no requiere recompilar nada, solo reiniciar
el servidor.

```json
{
  "sqlserver": {
    "host": "localhost",
    "port": "1433",
    "useWindowsAuth": false,
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
| `sqlserver.host`          | Dirección del servidor SQL (ej: `localhost`, una IP, o `localhost\SQLEXPRESS` para instancia nombrada) |
| `sqlserver.port`          | Puerto, normalmente `1433`. Déjalo como `""` (vacío) si usas una instancia nombrada (ver sección 4b) |
| `sqlserver.useWindowsAuth`| `true` para usar tu sesión de Windows en vez de usuario/password de SQL (ver sección 4c) |
| `sqlserver.user`          | Usuario de SQL Server (se ignora si `useWindowsAuth` es `true`) |
| `sqlserver.password`      | Contraseña (se ignora si `useWindowsAuth` es `true`) |
| `sqlserver.database`      | Nombre de la base de datos donde está tu tabla |
| `table.name`              | Nombre exacto de la tabla a usar |
| `table.keyColumn`         | Nombre de la columna llave primaria (debe ser un número entero) |

**Para cambiar de tabla rápidamente:** edita `table.name` y
`table.keyColumn`, guarda el archivo, y reinicia el servidor
(`Ctrl+C` y vuelve a correr `go run`). No hay que tocar ningún `.go`.

Si borras o dañas `config.json`, el programa no se cae: usa valores de
relleno (tabla `TuTabla`, columna `ID`, host `localhost`) y te avisa en
consola que debes crear el archivo.

## 4. Autenticación: SQL Server Auth vs Windows Auth

Hay dos formas de autenticarte contra SQL Server. Ambas se eligen y
configuran **exclusivamente en `config.json`**, con el campo
`useWindowsAuth`.

### 4a. Autenticación SQL Server (usuario y contraseña) — la más común

Es el modo por defecto (`useWindowsAuth: false`). Necesitas un usuario
y contraseña creados dentro de SQL Server (no de Windows).

```json
{
  "sqlserver": {
    "host": "localhost",
    "port": "1433",
    "useWindowsAuth": false,
    "user": "sa",
    "password": "TuPasswordReal",
    "database": "MiBaseDeDatos"
  },
  "table": {
    "name": "TuTabla",
    "keyColumn": "ID"
  }
}
```

**¿Dónde consigo el usuario y contraseña?**
- Si tú mismo instalaste SQL Server, el usuario `sa` (system
  administrator) y su contraseña son los que definiste durante la
  instalación.
- Si te dieron acceso a un SQL Server existente (trabajo, universidad,
  cliente), pide al administrador de la base de datos un usuario y
  contraseña de tipo "SQL Server Authentication" (no de Windows).
- Para verificar que un usuario/contraseña funcionan antes de tocar
  este proyecto, prueba conectarte con **SQL Server Management Studio
  (SSMS)** usando "SQL Server Authentication" con esos mismos datos.
  Si conecta ahí, conectará aquí también.

### 4b. Autenticación de Windows (Integrated Security / SSPI)

Se usa cuando el SQL Server está configurado para confiar en quien sea
que esté con sesión iniciada en Windows, sin pedir usuario/contraseña
de SQL. Es común en redes de oficina o dominio.

```json
{
  "sqlserver": {
    "host": "localhost",
    "port": "1433",
    "useWindowsAuth": true,
    "user": "",
    "password": "",
    "database": "MiBaseDeDatos"
  },
  "table": {
    "name": "TuTabla",
    "keyColumn": "ID"
  }
}
```

Con `useWindowsAuth: true`, deja `user` y `password` vacíos — se
ignoran. El programa usa automáticamente la sesión de Windows con la
que estés corriendo `go run` o el `.exe`.

**¿Cómo sé si debo usar este modo?**
- Abre **SQL Server Management Studio (SSMS)**, intenta conectarte
  eligiendo "Windows Authentication" (no "SQL Server Authentication").
  Si conecta sin pedirte usuario/contraseña, tu SQL Server soporta este
  modo y puedes usarlo aquí también.
- Si tu administrador de TI te dijo "usa tu usuario de dominio" o "se
  conecta solo, sin contraseña", es este modo.

**Requisitos para que funcione:**
- El programa debe correr en una máquina **Windows** (no en Linux/Mac,
  salvo que configures Kerberos por separado, lo cual está fuera del
  alcance de este proyecto).
- Tu usuario de Windows debe tener permisos otorgados explícitamente en
  ese SQL Server (esto lo configura el administrador de la base de
  datos, no tú desde este proyecto).
- Debes ejecutar `go run` o el `.exe` con la sesión de Windows correcta
  (la del usuario que sí tiene permiso). Si usas un usuario de Windows
  distinto, fallará aunque el código esté bien.

### 4c. ¿Dónde obtengo cada dato de conexión? (resumen rápido)

| Dato | Dónde lo consigues |
|------|---------------------|
| `host` | Te lo da quien administra el SQL Server. Si es tu propia PC, es `localhost`. Si es un servidor remoto, es su IP o nombre de red. |
| `port` | Ver sección 4d más abajo. |
| `database` | El nombre exacto de la base de datos, visible en SSMS bajo "Databases" al conectarte. |
| `user` / `password` | Ver sección 4a (solo aplica si `useWindowsAuth` es `false`). |
| `table.name` | El nombre exacto de la tabla, visible en SSMS bajo `Databases > [tu BD] > Tables`. |
| `table.keyColumn` | El nombre de la columna marcada con la llave 🔑 (Primary Key) en esa tabla dentro de SSMS. Debe ser de tipo numérico entero (`int`, `bigint`, etc.). |

### 4d. ¿Dónde obtengo el puerto?

El puerto por defecto de SQL Server es **1433**. Para confirmar el
tuyo:

1. Abre **SQL Server Configuration Manager** (búscalo en el menú inicio
   de Windows).
2. Ve a "SQL Server Network Configuration" → "Protocols for [tu
   instancia]".
3. Click derecho en "TCP/IP" → Properties → pestaña "IP Addresses" →
   baja hasta la sección "IPAll" → ahí ves el campo "TCP Port".

Si usas **SQL Server Express** con instancia nombrada (común en
instalaciones locales/de prueba, ej: `localhost\SQLEXPRESS`),
normalmente no hay puerto fijo. En ese caso:
- Pon en `host`: `localhost\SQLEXPRESS` (con tu nombre de instancia
  real, que ves en SSMS al conectarte)
- Deja `port` como `""` (vacío)

---

## 5. Requisitos e instalación

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

## 6. Ejecutar el servidor

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

## 7. Compilar un ejecutable (opcional)

```bash
go build -o scapegoat-server main.go tree.go config.go
./scapegoat-server        # Linux/Mac
scapegoat-server.exe      # Windows
```

`config.json` debe quedar en la misma carpeta que el ejecutable.

---

## 8. Errores comunes y solución

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

### Errores específicos al usar `useWindowsAuth: true`

**"Login failed for user" o "cannot generate SSPI context"**
Tu usuario de Windows actual no tiene permisos otorgados en ese SQL
Server. Pide al administrador de la base de datos que agregue tu
usuario de Windows (formato `DOMINIO\tuUsuario`) como login autorizado.

**Funciona en SSMS pero no en este programa**
Verifica que estés ejecutando `go run` o el `.exe` con la misma sesión
de Windows que usaste para conectar en SSMS. Si abriste una terminal
"Ejecutar como administrador" u otro usuario, puede estar usando una
identidad distinta.

**No funciona desde Linux/Mac**
Windows Auth vía SSPI solo funciona de forma nativa en Windows. En
Linux/Mac necesitarías Kerberos configurado aparte, lo cual no está
cubierto por esta configuración simple — en ese caso, usa autenticación
SQL Server (usuario/contraseña) en su lugar.

**El programa usa `sa`/`changeme` aunque puse `useWindowsAuth: true`**
Revisa que el JSON esté bien escrito (sin comas de más o de menos) y
que el campo se llame exactamente `useWindowsAuth` (con esa
capitalización). Si `config.json` tiene un error de formato, el
programa avisa en consola y usa los valores de relleno por defecto,
ignorando tu configuración real.

**"Server is not found or not accessible" con Windows Auth**
Confirma el nombre del host. Con instancia nombrada (ej:
`localhost\SQLEXPRESS`), asegúrate de escribir la barra invertida `\`
correctamente en el JSON — en JSON, una barra invertida sola puede dar
problemas; si tu editor la corrompe, prueba escribiendo el host como
`localhost\\SQLEXPRESS` (doble barra) como alternativa segura.

