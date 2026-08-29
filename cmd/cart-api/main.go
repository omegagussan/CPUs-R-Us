package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"

	"github.com/go-faster/errors"
	cartapi "github.com/omegagussan/cpus-r-us/pkg/cartapi"
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
func (h *CartHandler) resolveCart(ctx context.Context, userID string) (*cartapi.Cart, error) {
	h.mu.RLock()
	userCart, ok := h.carts[userID]
	h.mu.RUnlock()

	// If the user does not have a cart or it is empty, return an empty cart object.
	if !ok || len(userCart) == 0 {
		return &cartapi.Cart{
			Items:      []cartapi.CartItem{},
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

	var items []cartapi.CartItem
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
			items = append(items, cartapi.CartItem{
				ProductID: pair.productID,
				Quantity:  pair.quantity,
				Subtotal:  0.0,
			})
			continue
		}

		subtotal := float64(pair.quantity) * p.Price
		totalPrice += subtotal

		cartProd := cartapi.Product{
			ID:    p.ID,
			Name:  p.Name,
			Price: p.Price,
		}

		items = append(items, cartapi.CartItem{
			ProductID: pair.productID,
			Quantity:  pair.quantity,
			Product:   cartapi.NewOptProduct(cartProd),
			Subtotal:  subtotal,
		})
	}

	return &cartapi.Cart{
		Items:      items,
		TotalPrice: totalPrice,
	}, nil
}

// GetCart implements getCart operation.
func (h *CartHandler) GetCart(ctx context.Context, params cartapi.GetCartParams) (*cartapi.Cart, error) {
	return h.resolveCart(ctx, params.XUserID)
}

// ClearCart implements clearCart operation.
func (h *CartHandler) ClearCart(ctx context.Context, params cartapi.ClearCartParams) (*cartapi.Cart, error) {
	h.mu.Lock()
	delete(h.carts, params.XUserID)
	h.mu.Unlock()

	return h.resolveCart(ctx, params.XUserID)
}

// AddCartItem implements addCartItem operation.
func (h *CartHandler) AddCartItem(ctx context.Context, req *cartapi.AddCartItemRequest, params cartapi.AddCartItemParams) (cartapi.AddCartItemRes, error) {
	// 1. Verify the product exists in the Product API catalog.
	pRes, err := h.productClient.GetProduct(ctx, productapi.GetProductParams{ID: req.ProductID})
	if err != nil {
		return &cartapi.Error{
			Code:    http.StatusInternalServerError,
			Message: "failed to contact product service: " + err.Error(),
		}, nil
	}

	switch pRes.(type) {
	case *productapi.Product:
		// Product exists
	case *productapi.Error:
		return &cartapi.Error{
			Code:    http.StatusBadRequest,
			Message: "product " + req.ProductID + " does not exist",
		}, nil
	default:
		return &cartapi.Error{
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
		return &cartapi.Error{
			Code:    http.StatusInternalServerError,
			Message: "failed to resolve cart: " + err.Error(),
		}, nil
	}
	return cart, nil
}

// UpdateCartItem implements updateCartItem operation.
func (h *CartHandler) UpdateCartItem(ctx context.Context, req *cartapi.UpdateCartItemRequest, params cartapi.UpdateCartItemParams) (cartapi.UpdateCartItemRes, error) {
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
			return &cartapi.Error{
				Code:    http.StatusInternalServerError,
				Message: "failed to resolve cart: " + err.Error(),
			}, nil
		}
		return cart, nil
	}

	// Verify the product exists in the Product API.
	pRes, err := h.productClient.GetProduct(ctx, productapi.GetProductParams{ID: params.ProductID})
	if err != nil {
		return &cartapi.Error{
			Code:    http.StatusInternalServerError,
			Message: "failed to contact product service: " + err.Error(),
		}, nil
	}

	switch pRes.(type) {
	case *productapi.Product:
		// Product exists
	case *productapi.Error:
		return &cartapi.Error{
			Code:    http.StatusBadRequest,
			Message: "product " + params.ProductID + " does not exist",
		}, nil
	default:
		return &cartapi.Error{
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
		return &cartapi.Error{
			Code:    http.StatusInternalServerError,
			Message: "failed to resolve cart: " + err.Error(),
		}, nil
	}
	return cart, nil
}

// RemoveCartItem implements removeCartItem operation.
func (h *CartHandler) RemoveCartItem(ctx context.Context, params cartapi.RemoveCartItemParams) (*cartapi.Cart, error) {
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
func (h *CartHandler) NewError(ctx context.Context, err error) *cartapi.ErrorStatusCode {
	return &cartapi.ErrorStatusCode{
		StatusCode: http.StatusInternalServerError,
		Response: cartapi.Error{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		},
	}
}

// corsMiddleware wraps an http.Handler with standard CORS headers.
func corsMiddleware(next http.Handler) http.Handler {
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

	handler := NewCartHandler(productClient)
	srv, err := cartapi.NewServer(handler)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	// Wrap server with CORS middleware to enable easy frontend integration
	handlerWithCORS := corsMiddleware(srv)

	log.Printf("Cart API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, handlerWithCORS); err != nil {
		log.Fatalf("failed to listen and serve: %v", err)
	}
}
