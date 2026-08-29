package api

import (
	"context"
	"net/http"
	"sort"
	"sync"

	"github.com/go-faster/errors"
	productapi "github.com/omegagussan/cpus-r-us/pkg/productapi"
)

type CartHandler struct {
	mu            sync.RWMutex
	carts         map[string]map[string]int // userID -> productID -> quantity
	productClient *productapi.Client
}

func NewCartHandler(productClient *productapi.Client) *CartHandler {
	return &CartHandler{
		carts:         make(map[string]map[string]int),
		productClient: productClient,
	}
}

// resolveCart fetches product details from the Product API and resolves
// the full Cart details including titles, individual prices, subtotals, and total price.
func (h *CartHandler) resolveCart(ctx context.Context, userID string) (*Cart, error) {
	h.mu.RLock()
	userCart, ok := h.carts[userID]
	h.mu.RUnlock()

	// If the user does not have a cart or it is empty, return an empty cart object.
	if !ok || len(userCart) == 0 {
		return &Cart{
			Items:      []CartItem{},
			TotalPrice: 0.0,
		}, nil
	}

	// Fetch all products from Product API in one batch to resolve metadata and prices.
	products, err := h.productClient.ListProducts(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list products from Product API")
	}

	prodMap := make(map[string]productapi.Product)
	for _, p := range products {
		prodMap[p.ID] = p
	}

	var items []CartItem
	var totalPrice float64

	// Gather pairs and sort them to ensure deterministic listing order.
	type itemPair struct {
		productID string
		quantity  int
	}
	var pairs []itemPair
	h.mu.RLock()
	for pid, qty := range userCart {
		if qty > 0 {
			pairs = append(pairs, itemPair{pid, qty})
		}
	}
	h.mu.RUnlock()

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].productID < pairs[j].productID
	})

	for _, pair := range pairs {
		p, ok := prodMap[pair.productID]
		if !ok {
			// Product not found in Product API catalog.
			// Include it with 0 subtotal and without resolving details.
			items = append(items, CartItem{
				ProductID: pair.productID,
				Quantity:  pair.quantity,
				Subtotal:  0.0,
			})
			continue
		}

		subtotal := float64(pair.quantity) * p.Price
		totalPrice += subtotal

		cartProd := Product{
			ID:    p.ID,
			Name:  p.Name,
			Price: p.Price,
		}

		items = append(items, CartItem{
			ProductID: pair.productID,
			Quantity:  pair.quantity,
			Product:   NewOptProduct(cartProd),
			Subtotal:  subtotal,
		})
	}

	return &Cart{
		Items:      items,
		TotalPrice: totalPrice,
	}, nil
}

// GetCart implements getCart operation.
func (h *CartHandler) GetCart(ctx context.Context, params GetCartParams) (*Cart, error) {
	return h.resolveCart(ctx, params.XUserID)
}

// ClearCart implements clearCart operation.
func (h *CartHandler) ClearCart(ctx context.Context, params ClearCartParams) (*Cart, error) {
	h.mu.Lock()
	delete(h.carts, params.XUserID)
	h.mu.Unlock()

	return h.resolveCart(ctx, params.XUserID)
}

// AddCartItem implements addCartItem operation.
func (h *CartHandler) AddCartItem(ctx context.Context, req *AddCartItemRequest, params AddCartItemParams) (AddCartItemRes, error) {
	// 1. Verify the product exists in the Product API catalog.
	pRes, err := h.productClient.GetProduct(ctx, productapi.GetProductParams{ID: req.ProductID})
	if err != nil {
		return &Error{
			Code:    http.StatusInternalServerError,
			Message: "failed to contact product service: " + err.Error(),
		}, nil
	}

	switch pRes.(type) {
	case *productapi.Product:
		// Product exists
	case *productapi.Error:
		return &Error{
			Code:    http.StatusBadRequest,
			Message: "product " + req.ProductID + " does not exist",
		}, nil
	default:
		return &Error{
			Code:    http.StatusBadRequest,
			Message: "product not found",
		}, nil
	}

	// 2. Add or increment the quantity of the product.
	h.mu.Lock()
	if h.carts == nil {
		h.carts = make(map[string]map[string]int)
	}
	userCart, ok := h.carts[params.XUserID]
	if !ok {
		userCart = make(map[string]int)
		h.carts[params.XUserID] = userCart
	}
	userCart[req.ProductID] += req.Quantity
	h.mu.Unlock()

	// 3. Return the fully resolved cart.
	cart, err := h.resolveCart(ctx, params.XUserID)
	if err != nil {
		return &Error{
			Code:    http.StatusInternalServerError,
			Message: "failed to resolve cart: " + err.Error(),
		}, nil
	}
	return cart, nil
}

// UpdateCartItem implements updateCartItem operation.
func (h *CartHandler) UpdateCartItem(ctx context.Context, req *UpdateCartItemRequest, params UpdateCartItemParams) (UpdateCartItemRes, error) {
	// If quantity is 0, treat it as a removal.
	if req.Quantity == 0 {
		h.mu.Lock()
		if h.carts != nil {
			if userCart, ok := h.carts[params.XUserID]; ok {
				delete(userCart, params.ProductID)
			}
		}
		h.mu.Unlock()

		cart, err := h.resolveCart(ctx, params.XUserID)
		if err != nil {
			return &Error{
				Code:    http.StatusInternalServerError,
				Message: "failed to resolve cart: " + err.Error(),
			}, nil
		}
		return cart, nil
	}

	// Verify the product exists in the Product API.
	pRes, err := h.productClient.GetProduct(ctx, productapi.GetProductParams{ID: params.ProductID})
	if err != nil {
		return &Error{
			Code:    http.StatusInternalServerError,
			Message: "failed to contact product service: " + err.Error(),
		}, nil
	}

	switch pRes.(type) {
	case *productapi.Product:
		// Product exists
	case *productapi.Error:
		return &Error{
			Code:    http.StatusBadRequest,
			Message: "product " + params.ProductID + " does not exist",
		}, nil
	default:
		return &Error{
			Code:    http.StatusBadRequest,
			Message: "product not found",
		}, nil
	}

	// Set the new quantity.
	h.mu.Lock()
	if h.carts == nil {
		h.carts = make(map[string]map[string]int)
	}
	userCart, ok := h.carts[params.XUserID]
	if !ok {
		userCart = make(map[string]int)
		h.carts[params.XUserID] = userCart
	}
	userCart[params.ProductID] = req.Quantity
	h.mu.Unlock()

	cart, err := h.resolveCart(ctx, params.XUserID)
	if err != nil {
		return &Error{
			Code:    http.StatusInternalServerError,
			Message: "failed to resolve cart: " + err.Error(),
		}, nil
	}
	return cart, nil
}

// RemoveCartItem implements removeCartItem operation.
func (h *CartHandler) RemoveCartItem(ctx context.Context, params RemoveCartItemParams) (*Cart, error) {
	h.mu.Lock()
	if h.carts != nil {
		if userCart, ok := h.carts[params.XUserID]; ok {
			delete(userCart, params.ProductID)
		}
	}
	h.mu.Unlock()

	return h.resolveCart(ctx, params.XUserID)
}

// NewError creates *ErrorStatusCode from error returned by handler.
func (h *CartHandler) NewError(ctx context.Context, err error) *ErrorStatusCode {
	return &ErrorStatusCode{
		StatusCode: http.StatusInternalServerError,
		Response: Error{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		},
	}
}

// CorsMiddleware wraps an http.Handler with standard CORS headers.
func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-User-Id")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
