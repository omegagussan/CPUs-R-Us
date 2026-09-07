package tests

import (
	"testing"

	"github.com/google/uuid"
	cartapi "github.com/omegagussan/cpus-r-us/pkg/cartapi"
)

// TestUtils provides utility methods to build expected test objects and perform assertions.
type TestUtils struct{}

// NewCart constructs an expected cartapi.Cart object.
func (u TestUtils) NewCart(totalPrice float64, items ...cartapi.CartItem) cartapi.Cart {
	if items == nil {
		items = []cartapi.CartItem{}
	}
	return cartapi.Cart{
		Items:      items,
		TotalPrice: totalPrice,
	}
}

// NewCartItem constructs an expected cartapi.CartItem object.
func (u TestUtils) NewCartItem(productID uuid.UUID, quantity int, name string, price float64) cartapi.CartItem {
	subtotal := float64(quantity) * price
	var optProd cartapi.OptProduct
	if name != "" {
		optProd = cartapi.NewOptProduct(cartapi.Product{
			ID:    productID,
			Name:  name,
			Price: price,
		})
	}
	return cartapi.CartItem{
		ProductID: productID,
		Quantity:  quantity,
		Product:   optProd,
		Subtotal:  subtotal,
	}
}

// AssertCartEqual compares the actual cart against an expected cart.
func (u TestUtils) AssertCartEqual(t *testing.T, actual *cartapi.Cart, expected cartapi.Cart) {
	t.Helper()
	if actual == nil {
		t.Fatalf("expected cart, got nil")
	}
	if actual.TotalPrice != expected.TotalPrice {
		t.Errorf("expected total price %.2f, got %.2f", expected.TotalPrice, actual.TotalPrice)
	}
	if len(actual.Items) != len(expected.Items) {
		t.Fatalf("expected %d items in cart, got %d", len(expected.Items), len(actual.Items))
	}

	actMap := make(map[uuid.UUID]cartapi.CartItem)
	for _, actItem := range actual.Items {
		actMap[actItem.ProductID] = actItem
	}

	for _, expItem := range expected.Items {
		actItem, ok := actMap[expItem.ProductID]
		if !ok {
			t.Errorf("expected item with ProductID %s not found in cart", expItem.ProductID)
			continue
		}
		if actItem.Quantity != expItem.Quantity {
			t.Errorf("product %s: expected Quantity %d, got %d", expItem.ProductID, expItem.Quantity, actItem.Quantity)
		}
		if actItem.Subtotal != expItem.Subtotal {
			t.Errorf("product %s: expected Subtotal %.2f, got %.2f", expItem.ProductID, expItem.Subtotal, actItem.Subtotal)
		}
		if expItem.Product.Set {
			if !actItem.Product.Set {
				t.Errorf("product %s: expected Product to be set", expItem.ProductID)
			} else {
				expProd := expItem.Product.Value
				actProd := actItem.Product.Value
				if actProd.ID != expProd.ID || actProd.Name != expProd.Name || actProd.Price != expProd.Price {
					t.Errorf("product %s: expected Product %+v, got %+v", expItem.ProductID, expProd, actProd)
				}
			}
		}
	}
}
