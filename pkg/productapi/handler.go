package api

import (
	"context"
	"net/http"
	"sort"

	"github.com/google/uuid"
)

var (
	FloppyDiskID      = uuid.MustParse("da2140a8-b649-4171-874e-6e2a2254de85")
	PagerID           = uuid.MustParse("3a4b08f8-b3d2-4b21-9e2e-13c54d19cfc2")
	LaserDiskPlayerID = uuid.MustParse("be07b973-5a02-4638-89c0-9d0b046e7f7b")
)

type ProductHandler struct {
	products map[uuid.UUID]Product
}

func NewProductHandler() *ProductHandler {
	return &ProductHandler{
		products: map[uuid.UUID]Product{
			FloppyDiskID: {
				ID:    FloppyDiskID,
				Name:  "Floppy disk",
				Price: 10.0,
			},
			PagerID: {
				ID:    PagerID,
				Name:  "Pager",
				Price: 20.0,
			},
			LaserDiskPlayerID: {
				ID:    LaserDiskPlayerID,
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
	for _, p := range h.products {
		list = append(list, p)
	}
	// Sort deterministically by name
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
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
