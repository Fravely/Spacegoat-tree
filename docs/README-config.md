# README — config.go

## Qué es este archivo

Es el **puente entre `config.json` (texto editable) y el resto del
programa en Go**. Su trabajo es: leer `config.json`, validarlo,
rellenar lo que falte con valores de seguridad, y construir el
"connection string" (la cadena de texto que el driver de SQL Server
necesita para conectarse).

**Tampoco necesitas editar este archivo para uso normal** — toda la
personalización va en `config.json`. Solo lo tocarías si quisieras
agregar un campo de configuración nuevo que no existe todavía.

---

## La estructura `Config` (líneas 13–27)

```go
type Config struct {
	SQLServer struct {
		Host           string `json:"host"`
		Port           string `json:"port"`
		User           string `json:"user"`
		Password       string `json:"password"`
		Database       string `json:"database"`
		UseWindowsAuth bool   `json:"useWindowsAuth"`
	} `json:"sqlserver"`

	Table struct {
		Name      string `json:"name"`
		KeyColumn string `json:"keyColumn"`
	} `json:"table"`
}
```

Esto es un "molde" en Go que describe exactamente la forma que debe
tener `config.json`. Cada `json:"..."` le dice a Go qué nombre buscar
en el archivo JSON. Por ejemplo, `Host string \`json:"host"\`` significa
"busca en el JSON un campo llamado `host` (minúscula) y guárdalo aquí
como texto". Esta estructura es la razón por la que `config.json` debe
respetar esos nombres exactos.

`var appConfig Config` (línea 29) es la variable global donde queda
guardada la configuración una vez leída — el resto del programa
(`main.go`) la consulta como `appConfig.Table.Name`, etc.

---

## `loadConfig()` — Lee el archivo

```go
func loadConfig() {
	data, err := os.ReadFile("config.json")
	if err != nil {
		// No se encontró el archivo
		setDefaultConfig()
		return
	}

	if err := json.Unmarshal(data, &appConfig); err != nil {
		// El archivo existe pero el JSON está mal escrito
		setDefaultConfig()
		return
	}

	fillMissingDefaults()
	// imprime en consola qué se cargó
}
```

**Tres escenarios posibles, todos sin tumbar el programa:**

1. **`config.json` no existe** → se usan valores de relleno completos
   (`setDefaultConfig()`), con aviso claro en consola.
2. **`config.json` existe pero tiene un error de sintaxis** (coma de
   más, comilla faltante, etc.) → `json.Unmarshal` falla, se detecta el
   error, y también se usan valores de relleno completos.
3. **`config.json` existe y es válido, pero le faltan campos** (por
   ejemplo, olvidaste poner `"database"`) → se carga lo que sí hay, y
   `fillMissingDefaults()` rellena solo los campos vacíos uno por uno,
   sin descartar el resto de tu configuración.

`os.ReadFile("config.json")` busca el archivo **en la carpeta desde
donde se ejecuta el programa**, no necesariamente donde está
`main.go` — si ejecutas `go run` desde otra carpeta, no lo va a
encontrar (ver sección de errores más abajo).

---

## `setDefaultConfig()` — Valores de relleno totales

Se usa cuando no hay ningún `config.json` utilizable. Define:
- Host: `localhost`
- Puerto: `1433`
- Sin Windows Auth (`false`)
- Usuario: `sa`, contraseña: `changeme`
- Base de datos: `master`
- Tabla: `TuTabla`, columna: `ID`

Estos valores casi seguro **no van a conectar de verdad** (son
genéricos), pero permiten que el programa arranque igual y caiga al
modo "solo memoria con 500 números de prueba" sin errores fatales.

---

## `fillMissingDefaults()` — Relleno parcial, campo por campo

Revisa cada campo individualmente y solo lo rellena si está vacío
(`""`). Detalles importantes:

- **El puerto NO se fuerza** si lo dejaste vacío a propósito. Esto es
  intencional: si usas una instancia nombrada de SQL Server (ej:
  `localhost\SQLEXPRESS`), el puerto se resuelve automáticamente y
  forzarlo a `1433` rompería la conexión.
- **Usuario y contraseña solo se rellenan si `useWindowsAuth` es
  `false`.** Si activaste Windows Auth, esos campos se ignoran
  completamente — no hace falta llenarlos ni con relleno.

---

## `buildConnString()` — Arma la cadena de conexión real

Esta función construye el texto exacto que el driver de SQL Server
necesita para conectarse. Tiene dos caminos según `useWindowsAuth`:

**Con `useWindowsAuth: false` (usuario y contraseña):**
```go
return fmt.Sprintf("sqlserver://%s:%s@%s%s?database=%s",
	appConfig.SQLServer.User,
	appConfig.SQLServer.Password,
	appConfig.SQLServer.Host,
	port,
	appConfig.SQLServer.Database,
)
```
Produce algo como:
`sqlserver://sa:MiPassword@localhost:1433?database=MiBD`

**Con `useWindowsAuth: true` (autenticación de Windows):**
```go
return fmt.Sprintf("sqlserver://%s%s?database=%s&Integrated+Security=sspi",
	appConfig.SQLServer.Host,
	port,
	appConfig.SQLServer.Database,
)
```
Produce algo como:
`sqlserver://localhost:1433?database=MiBD&Integrated+Security=sspi`

Sin usuario ni contraseña en el texto — el driver usa la sesión de
Windows activa automáticamente.

---

## Qué puedes configurar aquí (y qué NO)

**Todo lo normal se configura en `config.json`, no aquí.** Lo único
que tendría sentido tocar en este archivo:

| Qué cambiar | Dónde | Por qué lo harías |
|---|---|---|
| Los valores de relleno por defecto | `setDefaultConfig()` | Si quieres que, sin `config.json`, el programa intente conectar a otro host/tabla por defecto |
| Agregar un campo nuevo de configuración | La estructura `Config` + ambas funciones de relleno | Si necesitas un dato adicional que `config.json` no contempla todavía |

---

## Errores específicos de este archivo

### `⚠️ No se encontró config.json, usando valores de relleno por defecto`
El archivo no está en la carpeta desde donde ejecutas `go run` o el
`.exe`. Verifica que `config.json` esté literalmente en la misma
carpeta que `main.go` (o junto al ejecutable si ya compilaste).

### `⚠️ config.json tiene un error de formato (...)`
El JSON está mal escrito. Causas típicas:
- Una coma de más al final del último campo de un bloque (en JSON
  estricto eso es inválido).
- Comillas faltantes alrededor de algún texto.
- Olvidar una llave `{` o `}` de cierre.

Recomendación: si editas `config.json` a mano, pégalo en un validador
de JSON online antes de guardarlo, o usa un editor de código (VS Code,
por ejemplo) que marque errores de sintaxis JSON automáticamente.

### El programa usa `sa` / `changeme` aunque sí edité `config.json`
Significa que cayó en uno de los dos casos de arriba. Revisa el
mensaje exacto que imprime la consola al arrancar — te dice cuál de
los dos pasó.

### Configuré `useWindowsAuth: true` pero sigue pidiendo usuario/contraseña
Verifica que el campo se llame exactamente `useWindowsAuth` (con esa
capitalización exacta) y que su valor sea `true` sin comillas (es un
booleano, no texto): `"useWindowsAuth": true`, no
`"useWindowsAuth": "true"`.
