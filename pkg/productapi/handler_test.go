package api

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestProductHandler_ListProducts(t *testing.T) {
	ctx := context.Background()
	handler := NewProductHandler()

	products, err := handler.ListProducts(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(products) != 3 {
		t.Errorf("expected 3 products, got %d", len(products))
	}

	expectedIDs := map[uuid.UUID]bool{
		FloppyDiskID:      true,
		PagerID:           true,
		LaserDiskPlayerID: true,
	}

	for _, p := range products {
		if !expectedIDs[p.ID] {
			t.Errorf("unexpected product ID: %s", p.ID)
		}
	}
}

func TestProductHandler_GetProduct_Success(t *testing.T) {
	ctx := context.Background()
	handler := NewProductHandler()

	res, err := handler.GetProduct(ctx, GetProductParams{ID: PagerID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prod, ok := res.(*Product)
	if !ok {
		t.Fatalf("expected *Product response, got %T", res)
	}

	if prod.Name != "Pager" || prod.Price != 20.0 {
		t.Errorf("unexpected product details: %+v", prod)
	}
}

func TestProductHandler_GetProduct_NotFound(t *testing.T) {
	ctx := context.Background()
	handler := NewProductHandler()

	nonExistent := uuid.New()
	res, err := handler.GetProduct(ctx, GetProductParams{ID: nonExistent})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	apiErr, ok := res.(*Error)
	if !ok {
		t.Fatalf("expected *Error response, got %T", res)
	}

	if apiErr.Code != 404 {
		t.Errorf("expected code 404, got %d", apiErr.Code)
	}
}
