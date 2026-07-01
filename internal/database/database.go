package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"scapegoat-tree/internal/config"

	_ "github.com/microsoft/go-mssqldb"
)

// Store encapsula operaciones SQL Server del proyecto.
type Store struct {
	DB     *sql.DB
	Config config.Config
}

// Connect intenta abrir conexión; retorna nil si falla.
func Connect(cfg config.Config) *Store {
	conn, err := sql.Open("sqlserver", cfg.ConnString())
	if err != nil {
		log.Printf("⚠️  No se pudo abrir conexión: %v\n", err)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		log.Printf("⚠️  No se pudo conectar (ping falló): %v\n", err)
		return nil
	}
	log.Println("✅ Conectado a SQL Server correctamente")
	return &Store{DB: conn, Config: cfg}
}

// FetchIDs obtiene claves de la tabla configurada o datos de prueba.
func (s *Store) FetchIDs() ([]int, error) {
	if s == nil || s.DB == nil {
		log.Println("⚠️  Sin conexión a BD configurada → usando datos de prueba (1-500)")
		return FallbackIDs(), nil
	}

	query := fmt.Sprintf("SELECT [%s] FROM [%s] ORDER BY [%s]",
		s.Config.Table.KeyColumn, s.Config.Table.Name, s.Config.Table.KeyColumn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		log.Printf("⚠️  Error al consultar la BD (%v) → usando datos de prueba (1-500)\n", err)
		return FallbackIDs(), nil
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			log.Printf("⚠️  Error leyendo fila (%v) → usando datos de prueba (1-500)\n", err)
			return FallbackIDs(), nil
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		log.Printf("⚠️  Error iterando resultados (%v) → usando datos de prueba (1-500)\n", err)
		return FallbackIDs(), nil
	}
	if len(ids) == 0 {
		log.Println("⚠️  La consulta no devolvió filas → usando datos de prueba (1-500)")
		return FallbackIDs(), nil
	}
	log.Printf("✅ %d IDs cargados desde la tabla [%s]\n", len(ids), s.Config.Table.Name)
	return ids, nil
}

// FetchRow devuelve una fila completa por clave.
func (s *Store) FetchRow(key int) (map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM [%s] WHERE [%s] = @p1",
		s.Config.Table.Name, s.Config.Table.KeyColumn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := s.DB.QueryContext(ctx, query, key)
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

// InsertKey inserta solo la columna clave.
func (s *Store) InsertKey(key int) error {
	query := fmt.Sprintf("INSERT INTO [%s] ([%s]) VALUES (@p1)",
		s.Config.Table.Name, s.Config.Table.KeyColumn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.DB.ExecContext(ctx, query, key)
	return err
}

// InsertFromPayload inserta con columnas adicionales del payload JSON.
func (s *Store) InsertFromPayload(key int, payload map[string]interface{}) error {
	columns := []string{s.Config.Table.KeyColumn}
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
		s.Config.Table.Name,
		"["+strings.Join(columns, "],[")+"]",
		strings.Join(placeholders, ","),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.DB.ExecContext(ctx, query, args...)
	return err
}

// DeleteKey elimina una fila por clave.
func (s *Store) DeleteKey(key int) error {
	query := fmt.Sprintf("DELETE FROM [%s] WHERE [%s] = @p1",
		s.Config.Table.Name, s.Config.Table.KeyColumn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.DB.ExecContext(ctx, query, key)
	return err
}

// TableColumns devuelve metadatos de columnas.
func (s *Store) TableColumns() ([]ColumnInfo, error) {
	query := fmt.Sprintf(`
		SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_NAME = '%s' AND TABLE_SCHEMA = 'dbo'
		ORDER BY ORDINAL_POSITION
	`, s.Config.Table.Name)

	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var colName, dataType, isNullable string
		if err := rows.Scan(&colName, &dataType, &isNullable); err != nil {
			continue
		}
		columns = append(columns, ColumnInfo{
			Name:       colName,
			DataType:   dataType,
			IsNullable: isNullable == "YES",
			IsKey:      strings.EqualFold(colName, s.Config.Table.KeyColumn),
		})
	}
	return columns, nil
}

// ColumnInfo describe una columna de la tabla configurada.
type ColumnInfo struct {
	Name       string `json:"name"`
	DataType   string `json:"dataType"`
	IsNullable bool   `json:"isNullable"`
	IsKey      bool   `json:"isKey"`
}

// FallbackIDs genera IDs 1..500 para pruebas sin BD.
func FallbackIDs() []int {
	ids := make([]int, 500)
	for i := 0; i < 500; i++ {
		ids[i] = i + 1
	}
	return ids
}
