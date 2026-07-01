package main

import (
	"embed"
	"log"
	"net"
	"os"

	"scapegoat-tree/internal/config"
	"scapegoat-tree/internal/database"
	"scapegoat-tree/internal/server"
)

//go:embed scapegoat-tree.html
var webFS embed.FS

func main() {
	cfg := config.Load()

	var store *database.Store
	dbEnabled := false
	if conn := database.Connect(cfg); conn != nil {
		store = conn
		dbEnabled = true
		defer store.DB.Close()
	} else {
		log.Println("⚠️  El servidor seguirá funcionando con datos de prueba (1-500) y sin sincronizar con la BD")
	}

	addr := ":8080"
	probe, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("⚠️  El puerto 8080 está ocupado (%v). Cierra el proceso anterior o cambia el puerto.", err)
		os.Exit(1)
	}
	probe.Close()

	srv := server.New(cfg, store, dbEnabled)
	log.Fatal(srv.Run(addr, webFS))
}
