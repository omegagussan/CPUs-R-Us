package api

import (
	"context"
	"net/http"
)

type ProductHandler struct {
	products map[string]Product
}

func NewProductHandler() *ProductHandler {
	return &ProductHandler{
		products: map[string]Product{
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
func (h *ProductHandler) GetProduct(ctx context.Context, params GetProductParams) (GetProductRes, error) {
	p, ok := h.products[params.ID]
	if !ok {
		return &Error{
			Code:    404,
			Message: "Product not found",
		}, nil
	}
	return &p, nil
}

// ListProducts implements listProducts operation.
func (h *ProductHandler) ListProducts(ctx context.Context) ([]Product, error) {
	list := make([]Product, 0, len(h.products))
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
func (h *ProductHandler) NewError(ctx context.Context, err error) *ErrorStatusCode {
	return &ErrorStatusCode{
		StatusCode: http.StatusInternalServerError,
		Response: Error{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		},
	}
}
