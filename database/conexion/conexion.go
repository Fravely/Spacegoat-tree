package main

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

type Sale struct {
	Invoice     string
	StockCode   string
	Description string
	Quantity    int
	InvoiceDate string
	Price       float64
	CustomerID  string
	Country     string
	Stock       int
}

func cargar_data(dbPath string) []Sale {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT Invoice, StockCode, Description, Quantity,
		       InvoiceDate, Price, "Customer ID", Country,
		       stock
		FROM productos
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var data []Sale

	for rows.Next() {
		var s Sale

		err := rows.Scan(
			&s.Invoice,
			&s.StockCode,
			&s.Description,
			&s.Quantity,
			&s.InvoiceDate,
			&s.Price,
			&s.CustomerID,
			&s.Country,
			&s.Stock,
		)
		if err != nil {
			log.Fatal(err)
		}

		data = append(data, s)
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return data
}