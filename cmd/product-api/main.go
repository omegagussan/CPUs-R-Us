package main

import (
	"context"
	"log"
	"net/http"
	"os"

	productapi "github.com/omegagussan/cpus-r-us/pkg/productapi"
)

type ProductHandler struct {
	products map[string]productapi.Product
}

func NewProductHandler() *ProductHandler {
	return &ProductHandler{
		products: map[string]productapi.Product{
			"floppy-disk": {
				ID:    "floppy-disk",
				Name:  "Floppy disk",
				Price: 10.0,
			},
			"pager": {
				ID:    "pager",
				Name:  "Pager",
				Price: 20.0,
			},
			"laser-disk-player": {
				ID:    "laser-disk-player",
				Name:  "Laser disk player",
				Price: 30.0,
			},
		},
	}
}

// GetProduct implements getProduct operation.
func (h *ProductHandler) GetProduct(ctx context.Context, params productapi.GetProductParams) (productapi.GetProductRes, error) {
	p, ok := h.products[params.ID]
	if !ok {
		return &productapi.Error{
			Code:    404,
			Message: "Product not found",
		}, nil
	}
	return &p, nil
}

// ListProducts implements listProducts operation.
func (h *ProductHandler) ListProducts(ctx context.Context) ([]productapi.Product, error) {
	list := make([]productapi.Product, 0, len(h.products))
	// Return products in a deterministic order
	order := []string{"floppy-disk", "pager", "laser-disk-player"}
	for _, id := range order {
		if p, ok := h.products[id]; ok {
			list = append(list, p)
		}
	}
	return list, nil
}

// NewError creates *ErrorStatusCode from error returned by handler.
func (h *ProductHandler) NewError(ctx context.Context, err error) *productapi.ErrorStatusCode {
	return &productapi.ErrorStatusCode{
		StatusCode: http.StatusInternalServerError,
		Response: productapi.Error{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		},
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	handler := NewProductHandler()
	srv, err := productapi.NewServer(handler)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	log.Printf("Product API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, srv); err != nil {
		log.Fatalf("failed to listen and serve: %v", err)
	}
}
