package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	productapi "github.com/omegagussan/cpus-r-us/pkg/productapi"
)

func TestCorsMiddleware_Options(t *testing.T) {
	// Simple dummy handler that should not be reached for OPTIONS
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("dummy handler should not be called on OPTIONS preflight request")
	})

	middleware := CorsMiddleware(dummyHandler)

	req := httptest.NewRequest("OPTIONS", "/cart", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, resp.StatusCode)
	}

	headers := []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
	}
	for _, h := range headers {
		if resp.Header.Get(h) == "" {
			t.Errorf("expected header %s to be set", h)
		}
	}
}

func TestCorsMiddleware_Forward(t *testing.T) {
	called := false
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := CorsMiddleware(dummyHandler)

	req := httptest.NewRequest("GET", "/cart", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	resp := w.Result()
	if !called {
		t.Error("expected dummy handler to be called")
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected Access-Control-Allow-Origin to be '*', got '%s'", resp.Header.Get("Access-Control-Allow-Origin"))
	}
}

func TestCartHandler_GetCart(t *testing.T) {
	ctx := context.Background()

	// Setup a mock Product API test server
	prodHandler := productapi.NewProductHandler()
	prodSrv, err := productapi.NewServer(prodHandler)
	if err != nil {
		t.Fatalf("failed to create product server: %v", err)
	}
	prodTestSrv := httptest.NewServer(prodSrv)
	defer prodTestSrv.Close()

	// Create Product Client
	prodClient, err := productapi.NewClient(prodTestSrv.URL)
	if err != nil {
		t.Fatalf("failed to create product client: %v", err)
	}

	// Create Cart Handler
	handler := NewCartHandler(prodClient)

	userID := "user-123"

	// 1. Get empty cart
	cart, err := handler.GetCart(ctx, GetCartParams{XUserID: userID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cart.Items) != 0 {
		t.Errorf("expected empty cart, got %d items", len(cart.Items))
	}

	// 2. Add an item manually to the in-memory map
	handler.mu.Lock()
	handler.carts[userID] = map[string]int{
		"floppy-disk": 3,
	}
	handler.mu.Unlock()

	// 3. Get cart again (should resolve product metadata)
	cart, err = handler.GetCart(ctx, GetCartParams{XUserID: userID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cart.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(cart.Items))
	}
	item := cart.Items[0]
	if item.ProductID != "floppy-disk" || item.Quantity != 3 || item.Subtotal != 30.0 {
		t.Errorf("unexpected item state: %+v", item)
	}
	if cart.TotalPrice != 30.0 {
		t.Errorf("expected total price 30.0, got %f", cart.TotalPrice)
	}
}

func TestCartHandler_AddCartItem_Validation(t *testing.T) {
	ctx := context.Background()

	// Setup mock Product API test server
	prodHandler := productapi.NewProductHandler()
	prodSrv, err := productapi.NewServer(prodHandler)
	if err != nil {
		t.Fatalf("failed to create product server: %v", err)
	}
	prodTestSrv := httptest.NewServer(prodSrv)
	defer prodTestSrv.Close()

	// Create Product Client
	prodClient, err := productapi.NewClient(prodTestSrv.URL)
	if err != nil {
		t.Fatalf("failed to create product client: %v", err)
	}

	// Create Cart Handler
	handler := NewCartHandler(prodClient)

	userID := "user-123"

	// 1. Add valid product
	res, err := handler.AddCartItem(ctx, &AddCartItemRequest{
		ProductID: "pager",
		Quantity:  2,
	}, AddCartItemParams{XUserID: userID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cart, ok := res.(*Cart)
	if !ok {
		t.Fatalf("expected *Cart response, got %T", res)
	}
	if len(cart.Items) != 1 || cart.Items[0].ProductID != "pager" || cart.Items[0].Quantity != 2 {
		t.Errorf("unexpected cart state: %+v", cart)
	}

	// 2. Add invalid product (should fail validation)
	resErr, err := handler.AddCartItem(ctx, &AddCartItemRequest{
		ProductID: "invalid-id",
		Quantity:  1,
	}, AddCartItemParams{XUserID: userID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	apiErr, ok := resErr.(*Error)
	if !ok {
		t.Fatalf("expected *Error response, got %T", resErr)
	}
	if apiErr.Code != 400 {
		t.Errorf("expected code 400, got %d", apiErr.Code)
	}
}
