package main

import (
	"log"
	"net/http"
	"os"

	productapi "github.com/omegagussan/cpus-r-us/pkg/productapi"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	handler := productapi.NewProductHandler()
	srv, err := productapi.NewServer(handler)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	log.Printf("Product API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, srv); err != nil {
		log.Fatalf("failed to listen and serve: %v", err)
	}
}
