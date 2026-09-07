package tests

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
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
	floppyID := productapi.FloppyDiskID
	pagerID := productapi.PagerID
	utils := TestUtils{}

	// Case A: Get cart for new user (should be empty).
	t.Run("GetEmptyCart", func(t *testing.T) {
		cart, err := cartClient.GetCart(ctx, cartapi.GetCartParams{XUserID: userID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedCart := utils.NewCart(0.0)
		utils.AssertCartEqual(t, cart, expectedCart)
	})

	// Case B: Add product "floppy-disk" (quantity: 2, price: 10 each, subtotal: 20).
	t.Run("AddFloppyDisk", func(t *testing.T) {
		res, err := cartClient.AddCartItem(ctx, &cartapi.AddCartItemRequest{
			ProductID: floppyID,
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

		expectedCart := utils.NewCart(20.0,
			utils.NewCartItem(floppyID, 2, "Floppy disk", 10.0),
		)
		utils.AssertCartEqual(t, cart, expectedCart)
	})

	// Case C: Add product "pager" (quantity: 1, price: 20, total: 40).
	t.Run("AddPager", func(t *testing.T) {
		res, err := cartClient.AddCartItem(ctx, &cartapi.AddCartItemRequest{
			ProductID: pagerID,
			Quantity:  1,
		}, cartapi.AddCartItemParams{XUserID: userID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cart, ok := res.(*cartapi.Cart)
		if !ok {
			t.Fatalf("unexpected response type: %T", res)
		}

		expectedCart := utils.NewCart(40.0,
			utils.NewCartItem(floppyID, 2, "Floppy disk", 10.0),
			utils.NewCartItem(pagerID, 1, "Pager", 20.0),
		)
		utils.AssertCartEqual(t, cart, expectedCart)
	})

	// Case D: Try to add non-existent product (should return 400 Bad Request).
	t.Run("AddNonExistentProduct", func(t *testing.T) {
		res, err := cartClient.AddCartItem(ctx, &cartapi.AddCartItemRequest{
			ProductID: uuid.New(),
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
			ProductID: floppyID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cart, ok := res.(*cartapi.Cart)
		if !ok {
			t.Fatalf("unexpected response type: %T", res)
		}

		expectedCart := utils.NewCart(70.0,
			utils.NewCartItem(floppyID, 5, "Floppy disk", 10.0),
			utils.NewCartItem(pagerID, 1, "Pager", 20.0),
		)
		utils.AssertCartEqual(t, cart, expectedCart)
	})

	// Case F: Delete pager from cart (only floppy-disk left, total: 50).
	t.Run("RemoveCartItem", func(t *testing.T) {
		cart, err := cartClient.RemoveCartItem(ctx, cartapi.RemoveCartItemParams{
			XUserID:   userID,
			ProductID: pagerID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedCart := utils.NewCart(50.0,
			utils.NewCartItem(floppyID, 5, "Floppy disk", 10.0),
		)
		utils.AssertCartEqual(t, cart, expectedCart)
	})

	// Case G: Clear cart (items: [], total: 0).
	t.Run("ClearCart", func(t *testing.T) {
		cart, err := cartClient.ClearCart(ctx, cartapi.ClearCartParams{XUserID: userID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedCart := utils.NewCart(0.0)
		utils.AssertCartEqual(t, cart, expectedCart)
	})
}
