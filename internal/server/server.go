package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"scapegoat-tree/internal/config"
	"scapegoat-tree/internal/database"
)

// Server expone la API REST y archivos estáticos.
type Server struct {
	cfg       config.Config
	db        *database.Store
	dbEnabled bool
	tree      *IntTree
	mu        sync.Mutex
	loadQueue []int
	loadIndex int
	loadTotal int
	dbMu      sync.RWMutex
}

// New crea el servidor con árbol vacío.
func New(cfg config.Config, db *database.Store, dbEnabled bool) *Server {
	return &Server{
		cfg:       cfg,
		db:        db,
		dbEnabled: dbEnabled,
		tree:      NewIntTree(),
	}
}

// TreeResponse es la respuesta estándar de operaciones sobre el árbol.
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

func (s *Server) treeResponse(rebalanced bool, scapegoatKey int, dbWarning string) TreeResponse {
	return TreeResponse{
		TreeNodes:    s.tree.Serialize(),
		Size:         s.tree.Size(),
		Height:       s.tree.Height(),
		Alpha:        s.tree.Alpha(),
		Rebalanced:   rebalanced,
		ScapegoatKey: scapegoatKey,
		DBWarning:    dbWarning,
	}
}

func (s *Server) treeResponseWithCount(rebalanceCount int, lastScapegoat int) TreeResponse {
	return TreeResponse{
		TreeNodes:      s.tree.Serialize(),
		Size:           s.tree.Size(),
		Height:         s.tree.Height(),
		Alpha:          s.tree.Alpha(),
		Rebalanced:     rebalanceCount > 0,
		RebalanceCount: rebalanceCount,
		ScapegoatKey:   lastScapegoat,
	}
}

func cors(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return false
	}
	return true
}

func (s *Server) isDBEnabled() bool {
	s.dbMu.RLock()
	defer s.dbMu.RUnlock()
	return s.dbEnabled
}

func (s *Server) handleTree(w http.ResponseWriter, r *http.Request) {
	if !cors(w, r) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	json.NewEncoder(w).Encode(s.treeResponse(false, 0, ""))
}

func (s *Server) handleInsert(w http.ResponseWriter, r *http.Request) {
	if !cors(w, r) {
		return
	}

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
		s.mu.Lock()
		defer s.mu.Unlock()

		inserted, rebal, scapegoatKey := s.tree.Insert(key)
		if !inserted {
			http.Error(w, `{"error":"la clave ya existe en el árbol"}`, 409)
			return
		}

		dbWarning := s.syncInsert(key, payload)
		if dbWarning == "rollback" {
			s.tree.Delete(key)
			http.Error(w, `{"error":"no se pudo insertar en la base de datos"}`, 500)
			return
		}
		json.NewEncoder(w).Encode(s.treeResponse(rebal, scapegoatKey, dbWarning))
		return
	}

	key, err := strconv.Atoi(r.URL.Query().Get("key"))
	if err != nil {
		http.Error(w, `{"error":"key inválido"}`, 400)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	inserted, rebal, scapegoatKey := s.tree.Insert(key)
	if !inserted {
		http.Error(w, `{"error":"la clave ya existe en el árbol"}`, 409)
		return
	}

	dbWarning := s.syncInsertKey(key)
	if dbWarning == "rollback" {
		s.tree.Delete(key)
		http.Error(w, `{"error":"no se pudo insertar en la base de datos"}`, 500)
		return
	}
	json.NewEncoder(w).Encode(s.treeResponse(rebal, scapegoatKey, dbWarning))
}

func (s *Server) syncInsert(key int, payload map[string]interface{}) string {
	if !s.isDBEnabled() {
		return "BD desactivada: el nodo solo se insertó en memoria."
	}
	if s.db == nil || s.db.DB == nil {
		return "Sin conexión a BD (db es nil), el nodo solo se insertó en memoria."
	}
	if err := s.db.InsertFromPayload(key, payload); err != nil {
		log.Printf("insert BD error: %v", err)
		return "rollback"
	}
	return ""
}

func (s *Server) syncInsertKey(key int) string {
	if !s.isDBEnabled() {
		return "BD desactivada: el nodo solo se insertó en memoria."
	}
	if s.db == nil || s.db.DB == nil {
		return "Sin conexión a BD: el nodo solo se insertó en memoria, no en la base de datos."
	}
	if err := s.db.InsertKey(key); err != nil {
		log.Printf("insert BD error: %v", err)
		return "rollback"
	}
	return ""
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if !cors(w, r) {
		return
	}
	key, err := strconv.Atoi(r.URL.Query().Get("key"))
	if err != nil {
		http.Error(w, `{"error":"key inválido"}`, 400)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	found, path := s.tree.Search(key)
	response := map[string]interface{}{
		"found": found,
		"path":  path,
	}

	if found && s.isDBEnabled() && s.db != nil && s.db.DB != nil {
		row, err := s.db.FetchRow(key)
		if err != nil {
			response["dbWarning"] = fmt.Sprintf("No se pudo traer la fila completa de la BD: %v", err)
		} else if row != nil {
			response["row"] = row
		}
	} else if found {
		response["dbWarning"] = "BD desactivada o sin conexión: no se pueden mostrar las demás columnas."
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	if !cors(w, r) {
		return
	}
	key, err := strconv.Atoi(r.URL.Query().Get("key"))
	if err != nil {
		http.Error(w, `{"error":"key inválido"}`, 400)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	dbWarning := ""
	if s.isDBEnabled() && s.db != nil && s.db.DB != nil {
		if err := s.db.DeleteKey(key); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"no se pudo eliminar de la base de datos: %v"}`, err), 500)
			return
		}
	} else {
		dbWarning = "BD desactivada o sin conexión: el nodo solo se eliminó en memoria, no en la base de datos."
	}

	deleted, rebal, scapegoatKey := s.tree.Delete(key)
	if !deleted {
		http.Error(w, `{"error":"nodo no encontrado en el árbol"}`, 404)
		return
	}
	json.NewEncoder(w).Encode(s.treeResponse(rebal, scapegoatKey, dbWarning))
}

func (s *Server) handleClear(w http.ResponseWriter, r *http.Request) {
	if !cors(w, r) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tree.Clear()
	s.loadQueue = nil
	s.loadIndex = 0
	json.NewEncoder(w).Encode(s.treeResponse(false, 0, ""))
}

func (s *Server) handleDBStatus(w http.ResponseWriter, r *http.Request) {
	if !cors(w, r) {
		return
	}
	s.dbMu.RLock()
	enabled := s.dbEnabled
	s.dbMu.RUnlock()
	json.NewEncoder(w).Encode(map[string]bool{"enabled": enabled})
}

func (s *Server) handleDBToggle(w http.ResponseWriter, r *http.Request) {
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
	s.dbMu.Lock()
	s.dbEnabled = req.Enabled
	s.dbMu.Unlock()
	log.Printf("🔄 Conexión a BD %s", map[bool]string{true: "activada", false: "desactivada"}[req.Enabled])
	json.NewEncoder(w).Encode(map[string]bool{"enabled": req.Enabled})
}

func (s *Server) handleTableColumns(w http.ResponseWriter, r *http.Request) {
	if !cors(w, r) {
		return
	}
	if s.db == nil || s.db.DB == nil {
		http.Error(w, `{"error":"Sin conexión a la base de datos"}`, 500)
		return
	}
	columns, err := s.db.TableColumns()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"No se pudo obtener información de columnas: %v"}`, err), 500)
		return
	}
	json.NewEncoder(w).Encode(columns)
}

func (s *Server) handleInitLoad(w http.ResponseWriter, r *http.Request) {
	if !cors(w, r) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tree.Clear()

	var ids []int
	var err error
	if s.db != nil {
		ids, err = s.db.FetchIDs()
	} else {
		ids = database.FallbackIDs()
	}
	if err != nil {
		http.Error(w, `{"error":"no se pudo conectar a la BD"}`, 500)
		return
	}
	s.loadQueue = ids
	s.loadIndex = 0
	s.loadTotal = len(ids)
	json.NewEncoder(w).Encode(map[string]int{"total": s.loadTotal})
}

func (s *Server) handleStepLoad(w http.ResponseWriter, r *http.Request) {
	if !cors(w, r) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loadIndex >= s.loadTotal {
		json.NewEncoder(w).Encode(map[string]interface{}{"done": true})
		return
	}
	key := s.loadQueue[s.loadIndex]
	s.loadIndex++
	_, rebal, scapegoatKey := s.tree.Insert(key)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"done":         false,
		"inserted":     key,
		"progress":     s.loadIndex,
		"rebalanced":   rebal,
		"scapegoatKey": scapegoatKey,
		"treeNodes":    s.tree.Serialize(),
		"size":         s.tree.Size(),
		"height":       s.tree.Height(),
	})
}

func (s *Server) handleBenchLoad(w http.ResponseWriter, r *http.Request) {
	if !cors(w, r) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tree.Clear()

	var ids []int
	var err error
	if s.db != nil {
		ids, err = s.db.FetchIDs()
	} else {
		ids = database.FallbackIDs()
	}
	if err != nil {
		http.Error(w, `{"error":"BD error"}`, 500)
		return
	}
	rebalances := 0
	lastScapegoat := 0
	for _, id := range ids {
		_, rebal, scapegoatKey := s.tree.Insert(id)
		if rebal {
			rebalances++
			lastScapegoat = scapegoatKey
		}
	}
	json.NewEncoder(w).Encode(s.treeResponseWithCount(rebalances, lastScapegoat))
}

// Run inicia el servidor HTTP en el puerto indicado y sirve la UI embebida.
func (s *Server) Run(addr string, webFS fs.FS) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/tree", s.handleTree)
	mux.HandleFunc("/insert", s.handleInsert)
	mux.HandleFunc("/search", s.handleSearch)
	mux.HandleFunc("/delete", s.handleDelete)
	mux.HandleFunc("/clear", s.handleClear)
	mux.HandleFunc("/init-load", s.handleInitLoad)
	mux.HandleFunc("/step-load", s.handleStepLoad)
	mux.HandleFunc("/load-bench", s.handleBenchLoad)
	mux.HandleFunc("/db-status", s.handleDBStatus)
	mux.HandleFunc("/db-toggle", s.handleDBToggle)
	mux.HandleFunc("/table-columns", s.handleTableColumns)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/scapegoat-tree.html" {
			http.NotFound(w, r)
			return
		}
		data, err := fs.ReadFile(webFS, "scapegoat-tree.html")
		if err != nil {
			http.Error(w, `{"error":"no se encontró scapegoat-tree.html"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	url := listenURL(addr)
	fmt.Printf("Scapegoat Tree  →  %s\n", url)
	go func() {
		time.Sleep(400 * time.Millisecond)
		openBrowser(url)
	}()

	return http.ListenAndServe(addr, mux)
}

func listenURL(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "http://localhost" + addr
	}
	if strings.Contains(addr, "://") {
		return addr
	}
	return "http://" + addr
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("No se pudo abrir el navegador automáticamente: %v", err)
		log.Printf("Abre manualmente: %s", url)
	}
}

// Close libera recursos del servidor.
func (s *Server) Close() {
	if s.db != nil && s.db.DB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.db.DB.Close()
		_ = ctx
	}
}
