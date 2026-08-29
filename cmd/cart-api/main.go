package main

import (
	"log"
	"net/http"
	"os"

	cartapi "github.com/omegagussan/cpus-r-us/pkg/cartapi"
	productapi "github.com/omegagussan/cpus-r-us/pkg/productapi"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	productAPIURL := os.Getenv("PRODUCT_API_URL")
	if productAPIURL == "" {
		productAPIURL = "http://localhost:8081"
	}

	log.Printf("Connecting to Product API at: %s", productAPIURL)
	productClient, err := productapi.NewClient(productAPIURL)
	if err != nil {
		log.Fatalf("failed to create product api client: %v", err)
	}

	handler := cartapi.NewCartHandler(productClient)
	srv, err := cartapi.NewServer(handler)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	// Wrap server with CORS middleware to enable easy frontend integration
	handlerWithCORS := cartapi.CorsMiddleware(srv)

	log.Printf("Cart API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, handlerWithCORS); err != nil {
		log.Fatalf("failed to listen and serve: %v", err)
	}
}
