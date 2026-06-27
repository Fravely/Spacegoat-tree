package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
)

// Config contains the SQL Server connection values used by the demo.
type Config struct {
	RawDSN   string
	Server   string
	Port     int
	Instance string
	Database string
	User     string
	Password string
	Encrypt  string
	Trust    bool
}

// ConfigFromEnv reads connection settings from environment variables.
func ConfigFromEnv() Config {
	var port int
	portValue := os.Getenv("SQLSERVER_PORT")
	port, err := strconv.Atoi(portValue)
	if err != nil {
		port = 0
	}
	return Config{
		RawDSN:   os.Getenv("SQLSERVER_DSN"),
		Server:   getenv("SQLSERVER_HOST", "localhost"),
		Port:     port,
		Instance: os.Getenv("SQLSERVER_INSTANCE"),
		Database: getenv("SQLSERVER_DATABASE", "InventarioProductosDB"),
		User:     getenv("SQLSERVER_USER", "sa"),
		Password: os.Getenv("SQLSERVER_PASSWORD"),
		Encrypt:  getenv("SQLSERVER_ENCRYPT", "disable"),
		Trust:    getenv("SQLSERVER_TRUST_CERT", "true") == "true",
	}
}

// DSN builds the sqlserver URL accepted by go-mssqldb.
func (c Config) DSN() string {
	if c.RawDSN != "" {
		return c.RawDSN
	}

	query := url.Values{}
	query.Set("database", c.Database)
	query.Set("encrypt", c.Encrypt)
	query.Set("TrustServerCertificate", strconv.FormatBool(c.Trust))

	host := c.Server
	if c.Port > 0 {
		host = fmt.Sprintf("%s:%d", c.Server, c.Port)
	}

	u := &url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(c.User, c.Password),
		Host:     host,
		Path:     c.Instance,
		RawQuery: query.Encode(),
	}
	return u.String()
}

// DSNForDatabase returns the same connection settings pointing to database.
func (c Config) DSNForDatabase(database string) string {
	if c.RawDSN != "" {
		return replaceDatabaseInDSN(c.RawDSN, database)
	}
	c.Database = database
	return c.DSN()
}

// Open ensures the target database exists, then creates and verifies a
// SQL Server connection to it.
func Open(ctx context.Context, cfg Config) (*sql.DB, error) {
	if err := EnsureDatabase(ctx, cfg); err != nil {
		return nil, err
	}
	return openRaw(ctx, cfg.DSN())
}

func openRaw(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(15 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// EnsureDatabase creates the configured database when it does not exist.
func EnsureDatabase(ctx context.Context, cfg Config) error {
	name := cfg.DatabaseName()
	if name == "" || strings.EqualFold(name, "master") {
		return nil
	}

	master, err := openRaw(ctx, cfg.DSNForDatabase("master"))
	if err != nil {
		return err
	}
	defer master.Close()

	var exists int
	if err := master.QueryRowContext(ctx,
		`SELECT CASE WHEN DB_ID(@name) IS NULL THEN 0 ELSE 1 END`,
		sql.Named("name", name),
	).Scan(&exists); err != nil {
		return err
	}
	if exists == 1 {
		return nil
	}

	_, err = master.ExecContext(ctx, `CREATE DATABASE `+quoteIdentifier(name))
	return err
}

// DatabaseName returns the configured database, including when RawDSN is used.
func (c Config) DatabaseName() string {
	if c.RawDSN != "" {
		if name := databaseFromDSN(c.RawDSN); name != "" {
			return name
		}
	}
	return c.Database
}

// EnsureSchema creates the Productos table when it does not exist.
func EnsureSchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
IF OBJECT_ID(N'dbo.Productos', N'U') IS NULL
BEGIN
	CREATE TABLE dbo.Productos (
		id INT NOT NULL PRIMARY KEY,
		nombre NVARCHAR(120) NOT NULL,
		categoria NVARCHAR(80) NOT NULL,
		precio DECIMAL(10, 2) NOT NULL,
		stock INT NOT NULL
	);
END`)
	return err
}

// SeedProducts inserts the sample dataset without overwriting existing rows.
func SeedProducts(ctx context.Context, db *sql.DB, products []Product) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
IF NOT EXISTS (SELECT 1 FROM dbo.Productos WHERE id = @id)
BEGIN
	INSERT INTO dbo.Productos (id, nombre, categoria, precio, stock)
	VALUES (@id, @name, @category, @price, @stock);
END`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, product := range products {
		_, err := stmt.ExecContext(
			ctx,
			sql.Named("id", product.ID),
			sql.Named("name", product.Name),
			sql.Named("category", product.Category),
			sql.Named("price", product.Price),
			sql.Named("stock", product.Stock),
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// UpsertProduct inserts a product or updates it when the ID already exists.
func UpsertProduct(ctx context.Context, db *sql.DB, product Product) error {
	_, err := db.ExecContext(ctx, `
MERGE dbo.Productos AS target
USING (SELECT @id AS id) AS source
ON target.id = source.id
WHEN MATCHED THEN
	UPDATE SET nombre = @name, categoria = @category, precio = @price, stock = @stock
WHEN NOT MATCHED THEN
	INSERT (id, nombre, categoria, precio, stock)
	VALUES (@id, @name, @category, @price, @stock);`,
		sql.Named("id", product.ID),
		sql.Named("name", product.Name),
		sql.Named("category", product.Category),
		sql.Named("price", product.Price),
		sql.Named("stock", product.Stock),
	)
	return err
}

// DeleteProduct removes a product by ID and reports whether a row was deleted.
func DeleteProduct(ctx context.Context, db *sql.DB, id int) (bool, error) {
	result, err := db.ExecContext(ctx, `DELETE FROM dbo.Productos WHERE id = @id`, sql.Named("id", id))
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

// ListProducts returns all products ordered by ID.
func ListProducts(ctx context.Context, db *sql.DB) ([]Product, error) {
	rows, err := db.QueryContext(ctx, `
SELECT id, nombre, categoria, precio, stock
FROM dbo.Productos
ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var product Product
		if err := rows.Scan(&product.ID, &product.Name, &product.Category, &product.Price, &product.Stock); err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return products, nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func databaseFromDSN(dsn string) string {
	if strings.HasPrefix(strings.ToLower(dsn), "sqlserver://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return ""
		}
		return u.Query().Get("database")
	}

	for _, part := range strings.Split(dsn, ";") {
		key, value, found := strings.Cut(part, "=")
		if !found {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "database", "initial catalog":
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func replaceDatabaseInDSN(dsn, database string) string {
	if strings.HasPrefix(strings.ToLower(dsn), "sqlserver://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return dsn
		}
		query := u.Query()
		query.Set("database", database)
		u.RawQuery = query.Encode()
		return u.String()
	}

	parts := strings.Split(dsn, ";")
	replaced := false
	for i, part := range parts {
		key, _, found := strings.Cut(part, "=")
		if !found {
			continue
		}
		normalized := strings.ToLower(strings.TrimSpace(key))
		if normalized == "database" || normalized == "initial catalog" {
			parts[i] = key + "=" + database
			replaced = true
		}
	}
	if !replaced {
		parts = append(parts, "database="+database)
	}
	return strings.Join(parts, ";")
}

func quoteIdentifier(name string) string {
	return "[" + strings.ReplaceAll(name, "]", "]]") + "]"
}
