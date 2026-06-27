package database

// Product is the sample domain record stored in SQL Server and indexed in the
// Scapegoat Tree by ID.
type Product struct {
	ID       int
	Name     string
	Category string
	Price    float64
	Stock    int
}

// SampleProducts returns deterministic seed data for demos and presentations.
func SampleProducts() []Product {
	return []Product{
		{ID: 101, Name: "Laptop Lenovo IdeaPad", Category: "Computo", Price: 2499.90, Stock: 8},
		{ID: 102, Name: "Mouse Logitech M170", Category: "Accesorios", Price: 49.90, Stock: 35},
		{ID: 103, Name: "Monitor LG 24", Category: "Computo", Price: 699.00, Stock: 12},
		{ID: 104, Name: "Teclado Redragon Kumara", Category: "Accesorios", Price: 189.90, Stock: 20},
		{ID: 105, Name: "SSD Kingston 1TB", Category: "Almacenamiento", Price: 319.90, Stock: 16},
		{ID: 106, Name: "Memoria RAM 16GB", Category: "Componentes", Price: 229.90, Stock: 18},
		{ID: 107, Name: "Webcam Logitech C920", Category: "Accesorios", Price: 299.00, Stock: 9},
		{ID: 108, Name: "Router TP-Link Archer", Category: "Redes", Price: 159.90, Stock: 14},
	}
}
