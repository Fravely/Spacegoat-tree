# README — config.json

## Qué es este archivo

**Este es el único archivo que necesitas editar para personalizar el
proyecto** — tabla, columna, credenciales, tipo de autenticación. Es
texto plano en formato JSON, no requiere saber programar ni
recompilar nada. Lo lee `config.go` cada vez que arranca el servidor
(`go run` o el `.exe`).

Debe estar **en la misma carpeta** que `main.go` (o junto al
ejecutable, si ya compilaste con `go build`).

---

## Contenido completo, campo por campo

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

### Bloque `sqlserver`

| Campo | Tipo | Qué es | Dónde lo consigues |
|---|---|---|---|
| `host` | texto | Dirección del servidor SQL | `localhost` si SQL Server está en tu misma PC. Una IP o nombre de red si es remoto. `localhost\NOMBREINSTANCIA` si usas instancia nombrada (común en SQL Server Express) |
| `port` | texto | Puerto de conexión | `"1433"` es el estándar. Déjalo `""` (vacío) si usas instancia nombrada. Ver "Cómo obtener el puerto" más abajo |
| `useWindowsAuth` | booleano (`true`/`false`, sin comillas) | Si es `true`, usa tu sesión de Windows en vez de usuario/contraseña | Ver sección de autenticación más abajo |
| `user` | texto | Usuario de SQL Server | Se ignora si `useWindowsAuth` es `true`. Si tú instalaste SQL Server, normalmente es `sa` |
| `password` | texto | Contraseña de ese usuario | Se ignora si `useWindowsAuth` es `true` |
| `database` | texto | Nombre de la base de datos donde está tu tabla | Visible en SQL Server Management Studio (SSMS), bajo "Databases" |

### Bloque `table`

| Campo | Tipo | Qué es | Dónde lo consigues |
|---|---|---|---|
| `name` | texto | Nombre exacto de la tabla a usar como árbol | En SSMS: `Databases > [tu BD] > Tables` |
| `keyColumn` | texto | Columna llave primaria (debe ser numérica entera) | En SSMS, la columna marcada con el ícono de llave 🔑 dentro de la tabla |

---

## Cómo cambiar de tabla rápidamente

1. Abre `config.json` con cualquier editor de texto (Bloc de notas
   sirve, aunque se recomienda VS Code o Notepad++ porque resaltan
   errores de sintaxis).
2. Cambia `table.name` y `table.keyColumn` por los de la nueva tabla.
3. Guarda el archivo.
4. Reinicia el servidor (`Ctrl+C` en la terminal donde corre, y vuelve
   a ejecutar `go run main.go tree.go config.go`).

No hace falta tocar ningún archivo `.go` ni recompilar nada.

---

## Cómo obtener el puerto

El puerto por defecto de SQL Server es **1433**. Para confirmarlo en
tu instalación:

1. Abre **SQL Server Configuration Manager** (búscalo en el menú
   inicio de Windows).
2. Ve a "SQL Server Network Configuration" → "Protocols for
   [tu instancia]".
3. Click derecho en "TCP/IP" → Properties → pestaña "IP Addresses" →
   baja hasta "IPAll" → ahí está el campo "TCP Port".

Si usas **SQL Server Express con instancia nombrada** (ej:
`localhost\SQLEXPRESS`), normalmente no hay puerto fijo. En ese caso:
```json
"host": "localhost\\SQLEXPRESS",
"port": "",
```
(la doble barra invertida `\\` es necesaria en JSON, porque una sola
barra `\` se interpreta como un carácter especial de escape).

---

## Cómo elegir el tipo de autenticación

### Opción A: usuario y contraseña de SQL Server (la más común)

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

¿De dónde sacas el usuario/contraseña? Si instalaste SQL Server tú
mismo, son los que definiste durante la instalación (usuario `sa` por
defecto). Si te dieron acceso a un servidor existente, pide al
administrador un usuario de tipo "SQL Server Authentication".

**Para verificar que funcionan antes de usarlos aquí:** abre SQL
Server Management Studio (SSMS), elige "SQL Server Authentication" e
intenta conectar con esos mismos datos. Si conecta ahí, conectará
aquí.

### Opción B: autenticación de Windows (sin contraseña)

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

Con `useWindowsAuth: true`, deja `user` y `password` como texto vacío
— se ignoran completamente. El programa usa automáticamente la cuenta
de Windows con la que estés ejecutando `go run` o el `.exe`.

**¿Cuándo usar esto?** Cuando tu SQL Server está configurado para
confiar en cuentas de Windows (típico en redes de oficina/dominio). Lo
sabes si en SSMS puedes conectar eligiendo "Windows Authentication"
sin que te pida contraseña.

**Importante:** este modo solo funciona si ejecutas el programa en
Windows, y si tu usuario de Windows actual tiene permisos otorgados en
ese SQL Server (eso lo da el administrador de la base de datos, no se
configura desde aquí).

---

## Qué pasa si el archivo no existe o está mal escrito

El programa **no se cae**. Si `config.json` falta, o tiene un error de
sintaxis (coma de más, comilla faltante, etc.), el servidor arranca de
todas formas usando valores de relleno genéricos, y avisa claramente
en la consola qué pasó. En ese estado, lo más probable es que no logre
conectar a ninguna base de datos real, y caerá automáticamente al modo
"solo memoria con 500 números de prueba" — sigue siendo usable para
ver el árbol funcionar, solo que sin datos reales.

---

## Errores comunes específicos de este archivo

### Errores de sintaxis JSON
JSON es estricto. Errores típicos al editar a mano:
- Coma de más después del último campo de un bloque:
  ```json
  "database": "MiBD",   ← esta coma sobra si es el último campo
  }
  ```
- Olvidar comillas alrededor de un texto:
  ```json
  "host": localhost   ← falta comillas: "host": "localhost"
  ```
- `useWindowsAuth` con comillas (debe ser booleano, sin comillas):
  ```json
  "useWindowsAuth": "true"   ← incorrecto
  "useWindowsAuth": true     ← correcto
  ```

**Recomendación:** pega el contenido en un validador de JSON online
antes de guardar, o edítalo con VS Code, que resalta estos errores
automáticamente con subrayado rojo.

### "El programa siempre usa los 500 números de prueba aunque edité esto"
Revisa el mensaje impreso en consola al arrancar — indica la causa
exacta (archivo no encontrado, error de sintaxis, credenciales
incorrectas, tabla inexistente, etc.). Ver también `README-config.md`
y `README-main.md` para detalles de cada posible causa.

### La barra invertida en instancias nombradas
Si tu host es `localhost\SQLEXPRESS`, en JSON debes escribirlo como
`localhost\\SQLEXPRESS` (doble barra), porque una sola barra invertida
en JSON es un carácter de escape especial y puede corromper el
archivo o no leerse como esperas.
