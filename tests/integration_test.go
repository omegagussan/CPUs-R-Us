package tests

import (
	"context"
	"net/http/httptest"
	"testing"

	cartapi "github.com/omegagussan/cpus-r-us/pkg/cartapi"
	productapi "github.com/omegagussan/cpus-r-us/pkg/productapi"
)

func TestIntegration(t *testing.T) {
	ctx := context.Background()

	// 1. Start Product API test server.
	prodHandler := productapi.NewProductHandler()
	prodSrv, err := productapi.NewServer(prodHandler)
	if err != nil {
		t.Fatalf("failed to create product server: %v", err)
	}
	prodTestSrv := httptest.NewServer(prodSrv)
	defer prodTestSrv.Close()

	// 2. Create Product Client pointing to the Product API test server.
	prodClient, err := productapi.NewClient(prodTestSrv.URL)
	if err != nil {
		t.Fatalf("failed to create product client: %v", err)
	}

	// 3. Start Cart API test server, injecting the Product Client.
	cartHandler := cartapi.NewCartHandler(prodClient)
	cartSrv, err := cartapi.NewServer(cartHandler)
	if err != nil {
		t.Fatalf("failed to create cart server: %v", err)
	}
	cartTestSrv := httptest.NewServer(cartSrv)
	defer cartTestSrv.Close()

	// 4. Create Cart Client pointing to the Cart API test server.
	cartClient, err := cartapi.NewClient(cartTestSrv.URL)
	if err != nil {
		t.Fatalf("failed to create cart client: %v", err)
	}

	userID := "test-user-1"

	// Case A: Get cart for new user (should be empty).
	t.Run("GetEmptyCart", func(t *testing.T) {
		cart, err := cartClient.GetCart(ctx, cartapi.GetCartParams{XUserID: userID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cart.Items) != 0 {
			t.Errorf("expected empty cart, got %d items", len(cart.Items))
		}
		if cart.TotalPrice != 0.0 {
			t.Errorf("expected total price 0.0, got %f", cart.TotalPrice)
		}
	})

	// Case B: Add product "floppy-disk" (quantity: 2, price: 10 each, subtotal: 20).
	t.Run("AddFloppyDisk", func(t *testing.T) {
		res, err := cartClient.AddCartItem(ctx, &cartapi.AddCartItemRequest{
			ProductID: "floppy-disk",
			Quantity:  2,
		}, cartapi.AddCartItemParams{XUserID: userID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cart, ok := res.(*cartapi.Cart)
		if !ok {
			if apiErr, ok := res.(*cartapi.Error); ok {
				t.Fatalf("API returned error: %s (code %d)", apiErr.Message, apiErr.Code)
			}
			t.Fatalf("unexpected response type: %T", res)
		}

		if len(cart.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(cart.Items))
		}
		item := cart.Items[0]
		if item.ProductID != "floppy-disk" {
			t.Errorf("expected floppy-disk, got %s", item.ProductID)
		}
		if item.Quantity != 2 {
			t.Errorf("expected quantity 2, got %d", item.Quantity)
		}
		if !item.Product.Set {
			t.Errorf("expected product to be set")
		} else {
			prod := item.Product.Value
			if prod.ID != "floppy-disk" || prod.Name != "Floppy disk" || prod.Price != 10.0 {
				t.Errorf("incorrect product details: %+v", prod)
			}
		}
		if item.Subtotal != 20.0 {
			t.Errorf("expected subtotal 20.0, got %f", item.Subtotal)
		}
		if cart.TotalPrice != 20.0 {
			t.Errorf("expected total price 20.0, got %f", cart.TotalPrice)
		}
	})

	// Case C: Add product "pager" (quantity: 1, price: 20, total: 40).
	t.Run("AddPager", func(t *testing.T) {
		res, err := cartClient.AddCartItem(ctx, &cartapi.AddCartItemRequest{
			ProductID: "pager",
			Quantity:  1,
		}, cartapi.AddCartItemParams{XUserID: userID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cart, ok := res.(*cartapi.Cart)
		if !ok {
			t.Fatalf("unexpected response type: %T", res)
		}

		// Items should be sorted: floppy-disk then pager
		if len(cart.Items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(cart.Items))
		}
		if cart.Items[0].ProductID != "floppy-disk" || cart.Items[1].ProductID != "pager" {
			t.Errorf("expected sorted items, got first=%s second=%s", cart.Items[0].ProductID, cart.Items[1].ProductID)
		}
		if cart.TotalPrice != 40.0 {
			t.Errorf("expected total price 40.0, got %f", cart.TotalPrice)
		}
	})

	// Case D: Try to add non-existent product (should return 400 Bad Request).
	t.Run("AddNonExistentProduct", func(t *testing.T) {
		res, err := cartClient.AddCartItem(ctx, &cartapi.AddCartItemRequest{
			ProductID: "non-existent-item",
			Quantity:  1,
		}, cartapi.AddCartItemParams{XUserID: userID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		apiErr, ok := res.(*cartapi.Error)
		if !ok {
			t.Fatalf("expected error response, got %T", res)
		}
		if apiErr.Code != 400 {
			t.Errorf("expected code 400, got %d", apiErr.Code)
		}
	})

	// Case E: Update floppy-disk quantity to 5 (subtotal: 50, total: 70).
	t.Run("UpdateQuantity", func(t *testing.T) {
		res, err := cartClient.UpdateCartItem(ctx, &cartapi.UpdateCartItemRequest{
			Quantity: 5,
		}, cartapi.UpdateCartItemParams{
			XUserID:   userID,
			ProductID: "floppy-disk",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cart, ok := res.(*cartapi.Cart)
		if !ok {
			t.Fatalf("unexpected response type: %T", res)
		}

		if len(cart.Items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(cart.Items))
		}
		// floppy disk is first
		fdItem := cart.Items[0]
		if fdItem.ProductID != "floppy-disk" || fdItem.Quantity != 5 || fdItem.Subtotal != 50.0 {
			t.Errorf("unexpected floppy disk state: %+v", fdItem)
		}
		if cart.TotalPrice != 70.0 {
			t.Errorf("expected total price 70.0, got %f", cart.TotalPrice)
		}
	})

	// Case F: Delete pager from cart (only floppy-disk left, total: 50).
	t.Run("RemoveCartItem", func(t *testing.T) {
		cart, err := cartClient.RemoveCartItem(ctx, cartapi.RemoveCartItemParams{
			XUserID:   userID,
			ProductID: "pager",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(cart.Items) != 1 {
			t.Fatalf("expected 1 item left, got %d", len(cart.Items))
		}
		if cart.Items[0].ProductID != "floppy-disk" {
			t.Errorf("expected floppy-disk left, got %s", cart.Items[0].ProductID)
		}
		if cart.TotalPrice != 50.0 {
			t.Errorf("expected total price 50.0, got %f", cart.TotalPrice)
		}
	})

	// Case G: Clear cart (items: [], total: 0).
	t.Run("ClearCart", func(t *testing.T) {
		cart, err := cartClient.ClearCart(ctx, cartapi.ClearCartParams{XUserID: userID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(cart.Items) != 0 {
			t.Errorf("expected empty cart, got %d items", len(cart.Items))
		}
		if cart.TotalPrice != 0.0 {
			t.Errorf("expected total price 0.0, got %f", cart.TotalPrice)
		}
	})
}
