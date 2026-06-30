package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// Config representa todo lo editable del programa: credenciales de
// SQL Server y el nombre de la tabla/columna que se usa como árbol.
// Se carga desde config.json al arrancar.
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
		Name      string `json:"name"`      // nombre de la tabla
		KeyColumn string `json:"keyColumn"` // columna llave primaria (entero)
	} `json:"table"`
}

var appConfig Config

// loadConfig lee config.json desde la carpeta del ejecutable.
// Si el archivo no existe o tiene errores, el programa AVISA con
// claridad y usa valores de relleno (no se cae).
func loadConfig() {
	data, err := os.ReadFile("config.json")
	if err != nil {
		log.Println("⚠️  No se encontró config.json, usando valores de relleno por defecto.")
		log.Println("⚠️  Crea un archivo config.json junto a main.go para personalizar la conexión.")
		setDefaultConfig()
		return
	}

	if err := json.Unmarshal(data, &appConfig); err != nil {
		log.Printf("⚠️  config.json tiene un error de formato (%v), usando valores de relleno.\n", err)
		setDefaultConfig()
		return
	}

	// Si algún campo quedó vacío en el JSON, rellenar con default
	// para que el programa nunca truene por un campo faltante.
	fillMissingDefaults()

	log.Printf("✅ Configuración cargada: tabla=%q, columna=%q, host=%q, database=%q\n",
		appConfig.Table.Name, appConfig.Table.KeyColumn,
		appConfig.SQLServer.Host, appConfig.SQLServer.Database)
}

func setDefaultConfig() {
	appConfig.SQLServer.Host = "localhost"
	appConfig.SQLServer.Port = "1433"
	appConfig.SQLServer.UseWindowsAuth = false
	appConfig.SQLServer.User = "sa"
	appConfig.SQLServer.Password = "changeme"
	appConfig.SQLServer.Database = "master"
	appConfig.Table.Name = "TuTabla"
	appConfig.Table.KeyColumn = "ID"
}

func fillMissingDefaults() {
	if appConfig.SQLServer.Host == "" {
		appConfig.SQLServer.Host = "localhost"
	}
	// Nota: el puerto NO se fuerza a 1433 si el usuario lo dejó vacío
	// a propósito (por ejemplo, al usar una instancia nombrada como
	// "localhost\SQLEXPRESS", donde el puerto se resuelve solo).
	if !appConfig.SQLServer.UseWindowsAuth {
		if appConfig.SQLServer.User == "" {
			appConfig.SQLServer.User = "sa"
		}
		if appConfig.SQLServer.Password == "" {
			appConfig.SQLServer.Password = "changeme"
		}
	}
	if appConfig.SQLServer.Database == "" {
		appConfig.SQLServer.Database = "master"
	}
	if appConfig.Table.Name == "" {
		appConfig.Table.Name = "TuTabla"
	}
	if appConfig.Table.KeyColumn == "" {
		appConfig.Table.KeyColumn = "ID"
	}
}

// buildConnString arma el connection string de SQL Server usando la
// configuración cargada desde config.json.
//
// Si useWindowsAuth=true, se usa autenticación integrada de Windows
// (Integrated Security=sspi) en vez de usuario/password de SQL Server.
// Esto solo funciona si el programa corre en una máquina Windows con
// una sesión que tenga permisos en el SQL Server destino.
func buildConnString() string {
	port := ""
	if appConfig.SQLServer.Port != "" {
		port = ":" + appConfig.SQLServer.Port
	}

	if appConfig.SQLServer.UseWindowsAuth {
		// Formato: sqlserver://host:port?database=X&Integrated+Security=sspi
		return fmt.Sprintf("sqlserver://%s%s?database=%s&Integrated+Security=sspi",
			appConfig.SQLServer.Host,
			port,
			appConfig.SQLServer.Database,
		)
	}

	return fmt.Sprintf("sqlserver://%s:%s@%s%s?database=%s",
		appConfig.SQLServer.User,
		appConfig.SQLServer.Password,
		appConfig.SQLServer.Host,
		port,
		appConfig.SQLServer.Database,
	)
}
