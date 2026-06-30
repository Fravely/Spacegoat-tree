package main

import (
	"fmt"
)

func main() {
	data := cargar_data("supermarket.db")

	fmt.Println("Registros:", len(data))

	for i := 0; i < 3 && i < len(data); i++ {
		fmt.Println(data[i])
	}
}