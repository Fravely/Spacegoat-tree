package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	db "scapegoat-project/database"
	"scapegoat-project/scapegoat"
)

type appState struct {
	mu   sync.Mutex
	tree *scapegoat.Tree[int, db.Product]
	sql  *sql.DB
	mode string
}

type treeResponse struct {
	Root    *scapegoat.NodeSnapshot[int, db.Product] `json:"root"`
	Stats   scapegoat.Stats                          `json:"stats"`
	InOrder []scapegoat.Entry[int, db.Product]       `json:"inOrder"`
	Mode    string                                   `json:"mode"`
}

type messageResponse struct {
	Message string `json:"message"`
}

type insertResponse struct {
	Message string                     `json:"message"`
	Trace   scapegoat.InsertTrace[int] `json:"trace"`
}

func main() {
	state := &appState{}
	if err := state.init(); err != nil {
		log.Fatal(err)
	}
	defer state.close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/tree", state.handleTree)
	mux.HandleFunc("POST /api/reset", state.handleReset)
	mux.HandleFunc("POST /api/products", state.handleInsert)
	mux.HandleFunc("GET /api/products/{id}", state.handleSearch)
	mux.HandleFunc("DELETE /api/products/{id}", state.handleDelete)
	mux.Handle("/", http.FileServer(http.Dir("web")))

	addr := ":8080"
	log.Printf("simulacion disponible en http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func (s *appState) init() error {
	if os.Getenv("SQLSERVER_DSN") == "" {
		s.mode = "memoria"
		return s.resetFromProducts(db.SampleProducts())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := db.Open(ctx, db.ConfigFromEnv())
	if err != nil {
		log.Printf("SQL Server no disponible, usando modo memoria: %v", err)
		s.mode = "memoria"
		return s.resetFromProducts(db.SampleProducts())
	}

	if err := db.EnsureSchema(ctx, conn); err != nil {
		conn.Close()
		return err
	}
	if err := db.SeedProducts(ctx, conn, db.SampleProducts()); err != nil {
		conn.Close()
		return err
	}

	products, err := db.ListProducts(ctx, conn)
	if err != nil {
		conn.Close()
		return err
	}

	s.sql = conn
	s.mode = "SQL Server"
	return s.resetFromProducts(products)
}

func (s *appState) close() {
	if s.sql != nil {
		s.sql.Close()
	}
}

func (s *appState) reset() error {
	if s.sql == nil {
		return s.resetFromProducts(db.SampleProducts())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	products, err := db.ListProducts(ctx, s.sql)
	if err != nil {
		return err
	}
	return s.resetFromProducts(products)
}

func (s *appState) resetFromProducts(products []db.Product) error {
	tree, err := scapegoat.NewOrdered[int, db.Product](2.0 / 3.0)
	if err != nil {
		return err
	}
	for _, product := range products {
		tree.Insert(product.ID, product)
	}
	s.mu.Lock()
	s.tree = tree
	s.mu.Unlock()
	return nil
}

func (s *appState) handleTree(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	writeJSON(w, http.StatusOK, treeResponse{
		Root:    s.tree.Snapshot(),
		Stats:   s.tree.Stats(),
		InOrder: s.tree.InOrder(),
		Mode:    s.mode,
	})
}

func (s *appState) handleReset(w http.ResponseWriter, r *http.Request) {
	if err := s.reset(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, messageResponse{Message: "Arbol reiniciado con productos de ejemplo"})
}

func (s *appState) handleInsert(w http.ResponseWriter, r *http.Request) {
	var product db.Product
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if product.ID <= 0 || strings.TrimSpace(product.Name) == "" {
		writeError(w, http.StatusBadRequest, errors.New("id y nombre son obligatorios"))
		return
	}

	if s.sql != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		if err := db.UpsertProduct(ctx, s.sql, product); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}

	s.mu.Lock()
	inserted, trace := s.tree.InsertWithTrace(product.ID, product)
	s.mu.Unlock()

	if inserted {
		writeJSON(w, http.StatusCreated, insertResponse{
			Message: fmt.Sprintf("Producto %d insertado", product.ID),
			Trace:   trace,
		})
		return
	}
	writeJSON(w, http.StatusOK, insertResponse{
		Message: fmt.Sprintf("Producto %d actualizado", product.ID),
		Trace:   trace,
	})
}

func (s *appState) handleSearch(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	s.mu.Lock()
	product, found := s.tree.Search(id)
	s.mu.Unlock()
	if !found {
		writeError(w, http.StatusNotFound, fmt.Errorf("producto %d no encontrado", id))
		return
	}
	writeJSON(w, http.StatusOK, product)
}

func (s *appState) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if s.sql != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		deletedDB, err := db.DeleteProduct(ctx, s.sql, id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if !deletedDB {
			writeError(w, http.StatusNotFound, fmt.Errorf("producto %d no encontrado en SQL Server", id))
			return
		}
	}

	s.mu.Lock()
	deleted := s.tree.Delete(id)
	s.mu.Unlock()
	if !deleted {
		writeError(w, http.StatusNotFound, fmt.Errorf("producto %d no encontrado", id))
		return
	}
	writeJSON(w, http.StatusOK, messageResponse{Message: fmt.Sprintf("Producto %d eliminado", id)})
}

func pathID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return 0, errors.New("id invalido")
	}
	return id, nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write json: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, messageResponse{Message: err.Error()})
}
