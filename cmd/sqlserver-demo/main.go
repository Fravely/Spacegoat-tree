package main

import (
	"context"
	"fmt"
	"log"
	"time"

	db "scapegoat-project/database"
	"scapegoat-project/scapegoat"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := db.Open(ctx, db.ConfigFromEnv())
	if err != nil {
		log.Fatalf("conectar a SQL Server: %v", err)
	}
	defer conn.Close()

	if err := db.EnsureSchema(ctx, conn); err != nil {
		log.Fatalf("crear esquema: %v", err)
	}
	if err := db.SeedProducts(ctx, conn, db.SampleProducts()); err != nil {
		log.Fatalf("cargar dataset: %v", err)
	}

	products, err := db.ListProducts(ctx, conn)
	if err != nil {
		log.Fatalf("leer productos: %v", err)
	}

	index, err := scapegoat.NewOrdered[int, db.Product](2.0 / 3.0)
	if err != nil {
		log.Fatal(err)
	}
	for _, product := range products {
		index.Insert(product.ID, product)
	}

	fmt.Printf("productos cargados desde SQL Server: %d\n", len(products))
	fmt.Printf("indice Scapegoat Tree: altura=%d, reconstrucciones=%d\n",
		index.Height(), index.Stats().Rebuilds)

	const searchID = 107
	product, found := index.Search(searchID)
	if !found {
		fmt.Printf("producto %d no encontrado\n", searchID)
		return
	}
	fmt.Printf("busqueda por arbol id=%d -> %s | %s | S/ %.2f | stock=%d\n",
		product.ID, product.Name, product.Category, product.Price, product.Stock)
}
