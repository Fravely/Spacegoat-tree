package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/microsoft/go-mssqldb"
)

//  ESTADO GLOBAL
var (
	tree       *ScapegoatTree
	db         *sql.DB
	mu         sync.Mutex
	loadQueue  []int
	loadIndex  int
	loadTotal  int
	dbEnabled  bool
	dbMu       sync.Mutex
)

//  RESPUESTA ESTÁNDAR
type TreeResponse struct {
	TreeNodes      []SerializedNode `json:"treeNodes"`
	Size           int              `json:"size"`
	Height         int              `json:"height"`
	Alpha          float64          `json:"alpha"`
	Rebalanced     bool             `json:"rebalanced"`
	RebalanceCount int              `json:"rebalanceCount"`
	ScapegoatKey   int              `json:"scapegoatKey"`
	DBWarning      string           `json:"dbWarning,omitempty"`
}

func treeResponse(rebalanced bool, scapegoatKey int, dbWarning string) TreeResponse {
	return TreeResponse{
		TreeNodes:    tree.Serialize(),
		Size:         tree.Size(),
		Height:       tree.Height(),
		Alpha:        tree.Alpha(),
		Rebalanced:   rebalanced,
		ScapegoatKey: scapegoatKey,
		DBWarning:    dbWarning,
	}
}

func treeResponseWithCount(rebalanceCount int, lastScapegoat int) TreeResponse {
	return TreeResponse{
		TreeNodes:      tree.Serialize(),
		Size:           tree.Size(),
		Height:         tree.Height(),
		Alpha:          tree.Alpha(),
		Rebalanced:     rebalanceCount > 0,
		RebalanceCount: rebalanceCount,
		ScapegoatKey:   lastScapegoat,
	}
}

// cors establece las cabeceras CORS y también maneja OPTIONS (preflight)
func cors(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return false // indica que la solicitud ya fue manejada (no continuar)
	}
	return true
}

//  HANDLERS
func handleTree(w http.ResponseWriter, r *http.Request) {
	if !cors(w, r) {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	json.NewEncoder(w).Encode(treeResponse(false, 0, ""))
}

func handleInsert(w http.ResponseWriter, r *http.Request) {
	if !cors(w, r) {
		return
	}

	// Verificar si es POST con JSON (modo BD activa)
	if r.Method == "POST" && r.Header.Get("Content-Type") == "application/json" {
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, `{"error":"JSON inválido"}`, 400)
			return
		}
		keyFloat, ok := payload["key"].(float64)
		if !ok {
			http.Error(w, `{"error":"key no encontrado o no es número"}`, 400)
			return
		}
		key := int(keyFloat)
		mu.Lock()
		defer mu.Unlock()

		inserted, rebal, scapegoatKey := tree.Insert(key)
		if !inserted {
			http.Error(w, `{"error":"la clave ya existe en el árbol"}`, 409)
			return
		}

		dbMu.Lock()
		enabled := dbEnabled
		dbMu.Unlock()
		dbWarning := ""

		if enabled && db != nil {
			if err := insertRowFromPayload(key, payload); err != nil {
				tree.Delete(key)
				http.Error(w, fmt.Sprintf(`{"error":"no se pudo insertar en la base de datos: %v"}`, err), 500)
				return
			}
		} else if enabled && db == nil {
			dbWarning = "Sin conexión a BD (db es nil), el nodo solo se insertó en memoria."
		} else {
			dbWarning = "BD desactivada: el nodo solo se insertó en memoria."
		}

		json.NewEncoder(w).Encode(treeResponse(rebal, scapegoatKey, dbWarning))
		return
	}

	// Modo solo memoria (GET o POST sin JSON)
	key, err := strconv.Atoi(r.URL.Query().Get("key"))
	if err != nil {
		http.Error(w, `{"error":"key inválido"}`, 400)
		return
	}
	mu.Lock()
	defer mu.Unlock()

	inserted, rebal, scapegoatKey := tree.Insert(key)
	if !inserted {
		http.Error(w, `{"error":"la clave ya existe en el árbol"}`, 409)
		return
	}

	dbMu.Lock()
	enabled := dbEnabled
	dbMu.Unlock()
	dbWarning := ""
	if enabled && db != nil {
		if err := insertRowInDB(key); err != nil {
			tree.Delete(key)
			http.Error(w, fmt.Sprintf(`{"error":"no se pudo insertar en la base de datos: %v"}`, err), 500)
			return
		}
	} else if enabled && db == nil {
		dbWarning = "Sin conexión a BD: el nodo solo se insertó en memoria, no en la base de datos."
	} else {
		dbWarning = "BD desactivada: el nodo solo se insertó en memoria."
	}

	json.NewEncoder(w).Encode(treeResponse(rebal, scapegoatKey, dbWarning))
}

// insertRowFromPayload construye y ejecuta un INSERT con todos los campos del mapa.
func insertRowFromPayload(key int, payload map[string]interface{}) error {
	columns := []string{appConfig.Table.KeyColumn}
	placeholders := []string{"@p1"}
	args := []interface{}{key}
	i := 2
	for col, val := range payload {
		if col == "key" {
			continue
		}
		columns = append(columns, col)
		placeholders = append(placeholders, fmt.Sprintf("@p%d", i))
		args = append(args, val)
		i++
	}
	query := fmt.Sprintf("INSERT INTO [%s] (%s) VALUES (%s)",
		appConfig.Table.Name,
		"["+strings.Join(columns, "],[")+"]",
		strings.Join(placeholders, ","),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	if !cors(w, r) {
		return
	}
	key, err := strconv.Atoi(r.URL.Query().Get("key"))
	if err != nil {
		http.Error(w, `{"error":"key inválido"}`, 400)
		return
	}
	mu.Lock()
	defer mu.Unlock()

	found, path := tree.Search(key)
	response := map[string]interface{}{
		"found": found,
		"path":  path,
	}

	dbMu.Lock()
	enabled := dbEnabled
	dbMu.Unlock()

	if found && enabled && db != nil {
		row, err := fetchRowFromDB(key)
		if err != nil {
			response["dbWarning"] = fmt.Sprintf("No se pudo traer la fila completa de la BD: %v", err)
		} else if row != nil {
			response["row"] = row
		}
	} else if found && (!enabled || db == nil) {
		response["dbWarning"] = "BD desactivada o sin conexión: no se pueden mostrar las demás columnas."
	}

	json.NewEncoder(w).Encode(response)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	if !cors(w, r) {
		return
	}
	key, err := strconv.Atoi(r.URL.Query().Get("key"))
	if err != nil {
		http.Error(w, `{"error":"key inválido"}`, 400)
		return
	}
	mu.Lock()
	defer mu.Unlock()

	dbMu.Lock()
	enabled := dbEnabled
	dbMu.Unlock()

	dbWarning := ""
	if enabled && db != nil {
		if err := deleteRowFromDB(key); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"no se pudo eliminar de la base de datos: %v"}`, err), 500)
			return
		}
	} else {
		dbWarning = "BD desactivada o sin conexión: el nodo solo se eliminó en memoria, no en la base de datos."
	}

	deleted, rebal, scapegoatKey := tree.Delete(key)
	if !deleted {
		http.Error(w, `{"error":"nodo no encontrado en el árbol"}`, 404)
		return
	}
	json.NewEncoder(w).Encode(treeResponse(rebal, scapegoatKey, dbWarning))
}

func handleClear(w http.ResponseWriter, r *http.Request) {
	if !cors(w, r) {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	tree.Clear()
	loadQueue = nil
	loadIndex = 0
	json.NewEncoder(w).Encode(treeResponse(false, 0, ""))
}

//  ESTADO DE LA BD
func handleDBStatus(w http.ResponseWriter, r *http.Request) {
	if !cors(w, r) {
		return
	}
	dbMu.Lock()
	enabled := dbEnabled
	dbMu.Unlock()
	json.NewEncoder(w).Encode(map[string]bool{"enabled": enabled})
}

func handleDBToggle(w http.ResponseWriter, r *http.Request) {
	if !cors(w, r) {
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"JSON inválido"}`, 400)
		return
	}
	dbMu.Lock()
	dbEnabled = req.Enabled
	dbMu.Unlock()
	log.Printf("🔄 Conexión a BD %s", map[bool]string{true: "activada", false: "desactivada"}[req.Enabled])
	json.NewEncoder(w).Encode(map[string]bool{"enabled": req.Enabled})
}

func handleTableColumns(w http.ResponseWriter, r *http.Request) {
	if !cors(w, r) {
		return
	}
	if db == nil {
		http.Error(w, `{"error":"Sin conexión a la base de datos"}`, 500)
		return
	}
	query := fmt.Sprintf(`
		SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_NAME = '%s' AND TABLE_SCHEMA = 'dbo'
		ORDER BY ORDINAL_POSITION
	`, appConfig.Table.Name)

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"No se pudo obtener información de columnas: %v"}`, err), 500)
		return
	}
	defer rows.Close()

	type ColumnInfo struct {
		Name       string `json:"name"`
		DataType   string `json:"dataType"`
		IsNullable bool   `json:"isNullable"`
		IsKey      bool   `json:"isKey"`
	}
	columns := []ColumnInfo{}
	for rows.Next() {
		var colName, dataType, isNullable string
		if err := rows.Scan(&colName, &dataType, &isNullable); err != nil {
			continue
		}
		isKey := strings.EqualFold(colName, appConfig.Table.KeyColumn)
		columns = append(columns, ColumnInfo{
			Name:       colName,
			DataType:   dataType,
			IsNullable: isNullable == "YES",
			IsKey:      isKey,
		})
	}
	json.NewEncoder(w).Encode(columns)
}


//  CARGA DEMO Y BENCHMARK

func handleInitLoad(w http.ResponseWriter, r *http.Request) {
	if !cors(w, r) {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	tree.Clear()
	ids, err := fetchIDsFromDB()
	if err != nil {
		http.Error(w, `{"error":"no se pudo conectar a la BD"}`, 500)
		return
	}
	loadQueue = ids
	loadIndex = 0
	loadTotal = len(ids)
	json.NewEncoder(w).Encode(map[string]int{"total": loadTotal})
}

func handleStepLoad(w http.ResponseWriter, r *http.Request) {
	if !cors(w, r) {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if loadIndex >= loadTotal {
		json.NewEncoder(w).Encode(map[string]interface{}{"done": true})
		return
	}
	key := loadQueue[loadIndex]
	loadIndex++
	_, rebal, scapegoatKey := tree.Insert(key)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"done":         false,
		"inserted":     key,
		"progress":     loadIndex,
		"rebalanced":   rebal,
		"scapegoatKey": scapegoatKey,
		"treeNodes":    tree.Serialize(),
		"size":         tree.Size(),
		"height":       tree.Height(),
	})
}

func handleBenchLoad(w http.ResponseWriter, r *http.Request) {
	if !cors(w, r) {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	tree.Clear()
	ids, err := fetchIDsFromDB()
	if err != nil {
		http.Error(w, `{"error":"BD error"}`, 500)
		return
	}
	rebalances := 0
	lastScapegoat := 0
	for _, id := range ids {
		_, rebal, scapegoatKey := tree.Insert(id)
		if rebal {
			rebalances++
			lastScapegoat = scapegoatKey
		}
	}
	json.NewEncoder(w).Encode(treeResponseWithCount(rebalances, lastScapegoat))
}
//  CONEXIÓN Y OPERACIONES SQL SERVER

func initDB() (*sql.DB, error) {
	connString := buildConnString()
	conn, err := sql.Open("sqlserver", connString)
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir conexión: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("no se pudo conectar (ping falló): %w", err)
	}
	return conn, nil
}

func fetchIDsFromDB() ([]int, error) {
	if db == nil {
		log.Println("⚠️  Sin conexión a BD configurada → usando datos de prueba (1-500)")
		return fallbackIDs(), nil
	}
	query := fmt.Sprintf("SELECT [%s] FROM [%s] ORDER BY [%s]",
		appConfig.Table.KeyColumn, appConfig.Table.Name, appConfig.Table.KeyColumn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("⚠️  Error al consultar la BD (%v) → usando datos de prueba (1-500)\n", err)
		return fallbackIDs(), nil
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			log.Printf("⚠️  Error leyendo fila (%v) → usando datos de prueba (1-500)\n", err)
			return fallbackIDs(), nil
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		log.Printf("⚠️  Error iterando resultados (%v) → usando datos de prueba (1-500)\n", err)
		return fallbackIDs(), nil
	}
	if len(ids) == 0 {
		log.Println("⚠️  La consulta no devolvió filas → usando datos de prueba (1-500)")
		return fallbackIDs(), nil
	}
	log.Printf("✅ %d IDs cargados desde la tabla [%s]\n", len(ids), appConfig.Table.Name)
	return ids, nil
}

func fetchRowFromDB(key int) (map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM [%s] WHERE [%s] = @p1",
		appConfig.Table.Name, appConfig.Table.KeyColumn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := db.QueryContext(ctx, query, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	if !rows.Next() {
		return nil, nil
	}
	values := make([]interface{}, len(columns))
	pointers := make([]interface{}, len(columns))
	for i := range values {
		pointers[i] = &values[i]
	}
	if err := rows.Scan(pointers...); err != nil {
		return nil, err
	}
	result := make(map[string]interface{})
	for i, col := range columns {
		v := values[i]
		if b, ok := v.([]byte); ok {
			result[col] = string(b)
		} else {
			result[col] = v
		}
	}
	return result, nil
}

func insertRowInDB(key int) error {
	query := fmt.Sprintf("INSERT INTO [%s] ([%s]) VALUES (@p1)",
		appConfig.Table.Name, appConfig.Table.KeyColumn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.ExecContext(ctx, query, key)
	return err
}

func deleteRowFromDB(key int) error {
	query := fmt.Sprintf("DELETE FROM [%s] WHERE [%s] = @p1",
		appConfig.Table.Name, appConfig.Table.KeyColumn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.ExecContext(ctx, query, key)
	return err
}

func fallbackIDs() []int {
	ids := make([]int, 500)
	for i := 0; i < 500; i++ {
		ids[i] = i + 1
	}
	return ids
}

//  MAIN

func main() {
	loadConfig()
	tree = NewScapegoatTree()

	conn, err := initDB()
	if err != nil {
		log.Printf("⚠️  No se pudo conectar a SQL Server: %v\n", err)
		log.Println("⚠️  El servidor seguirá funcionando con datos de prueba (1-500) y sin sincronizar con la BD")
		db = nil
		dbEnabled = false
	} else {
		db = conn
		defer db.Close()
		dbEnabled = true
		log.Println("✅ Conectado a SQL Server correctamente")
	}

	http.HandleFunc("/tree", handleTree)
	http.HandleFunc("/insert", handleInsert)
	http.HandleFunc("/search", handleSearch)
	http.HandleFunc("/delete", handleDelete)
	http.HandleFunc("/clear", handleClear)
	http.HandleFunc("/init-load", handleInitLoad)
	http.HandleFunc("/step-load", handleStepLoad)
	http.HandleFunc("/load-bench", handleBenchLoad)
	http.HandleFunc("/db-status", handleDBStatus)
	http.HandleFunc("/db-toggle", handleDBToggle)
	http.HandleFunc("/table-columns", handleTableColumns)

	http.Handle("/", http.FileServer(http.Dir("./")))

	fmt.Println("Scapegoat Tree  →  http://localhost:8080 ")
	log.Fatal(http.ListenAndServe(":8080", nil))
}