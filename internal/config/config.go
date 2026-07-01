package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Config representa credenciales SQL Server y metadatos de tabla.
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

// Load lee config.json o aplica valores por defecto.
func Load() Config {
	var cfg Config
	data, path, err := readConfigFile()
	if err != nil {
		log.Println("⚠️  No se encontró config.json, usando valores de relleno por defecto.")
		log.Println("⚠️  Crea un archivo config.json junto al ejecutable para personalizar la conexión.")
		setDefaults(&cfg)
		return cfg
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("⚠️  config.json tiene un error de formato (%v), usando valores de relleno.\n", err)
		setDefaults(&cfg)
		return cfg
	}

	fillMissing(&cfg)
	log.Printf("✅ Configuración cargada desde %q: tabla=%q, columna=%q, host=%q, database=%q\n",
		path, cfg.Table.Name, cfg.Table.KeyColumn,
		cfg.SQLServer.Host, cfg.SQLServer.Database)
	return cfg
}

func readConfigFile() ([]byte, string, error) {
	candidates := []string{"config.json"}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "config.json"))
	}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, path, nil
		}
	}
	return nil, "", os.ErrNotExist
}

func setDefaults(cfg *Config) {
	cfg.SQLServer.Host = "localhost"
	cfg.SQLServer.Port = "1433"
	cfg.SQLServer.UseWindowsAuth = false
	cfg.SQLServer.User = "sa"
	cfg.SQLServer.Password = "changeme"
	cfg.SQLServer.Database = "master"
	cfg.Table.Name = "TuTabla"
	cfg.Table.KeyColumn = "ID"
}

func fillMissing(cfg *Config) {
	if cfg.SQLServer.Host == "" {
		cfg.SQLServer.Host = "localhost"
	}
	if !cfg.SQLServer.UseWindowsAuth {
		if cfg.SQLServer.User == "" {
			cfg.SQLServer.User = "sa"
		}
		if cfg.SQLServer.Password == "" {
			cfg.SQLServer.Password = "changeme"
		}
	}
	if cfg.SQLServer.Database == "" {
		cfg.SQLServer.Database = "master"
	}
	if cfg.Table.Name == "" {
		cfg.Table.Name = "TuTabla"
	}
	if cfg.Table.KeyColumn == "" {
		cfg.Table.KeyColumn = "ID"
	}
}

// ConnString arma el connection string de SQL Server.
func (c Config) ConnString() string {
	port := ""
	if c.SQLServer.Port != "" {
		port = ":" + c.SQLServer.Port
	}

	if c.SQLServer.UseWindowsAuth {
		return fmt.Sprintf("sqlserver://%s%s?database=%s&Integrated+Security=sspi",
			c.SQLServer.Host, port, c.SQLServer.Database)
	}

	return fmt.Sprintf("sqlserver://%s:%s@%s%s?database=%s",
		c.SQLServer.User,
		c.SQLServer.Password,
		c.SQLServer.Host,
		port,
		c.SQLServer.Database,
	)
}
