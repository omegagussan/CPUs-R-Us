# CPUs-R-Us e-Commerce Backend PoC

A PoC backend implementing a Product Catalog API and a Shopping Cart API for CPUs-R-Us.

---

## 🚀 How to Run

1. **Start the Product API** (Port 8081):
   ```bash
   PORT=8081 go run ./cmd/product-api/main.go
   ```

2. **Start the Cart API** (Port 8082):
   ```bash
   PORT=8082 PRODUCT_API_URL=http://localhost:8081 go run ./cmd/cart-api/main.go
   ```

---

## 🧪 Quick Test

```bash
# 1. Fetch products
curl -s http://localhost:8081/products

# 2. Add 2 floppy disks to cart (requires X-User-Id header)
curl -s -X POST -H "X-User-Id: user-1" -H "Content-Type: application/json" \
  -d '{"product_id":"floppy-disk","quantity":2}' \
  http://localhost:8082/cart/items

# 3. View the cart (resolves product details and total price)
curl -s -H "X-User-Id: user-1" http://localhost:8082/cart
```

---

## 📐 Design Decisions

* **Microservices**: Split into two services (`Product API` & `Cart API`) simulating production microservice boundaries.
* **OpenAPI Generator (`ogen`)**: Used Go code-generation from `/api/*.yaml` contracts to enforce type safety.
* **Client-Side Resolution**: The Cart API acts as an aggregator. It consumes the Product API at runtime via a generated Go HTTP client to resolve product metadata and calculate prices.
* **CORS Middleware**: Implemented standard CORS headers on the Cart API to allow direct integration with frontend websites.
* **In-Memory Store**: Cart items are stored in a thread-safe Go map for simplicity in this PoC.

---

## 🧪 Running Integration Tests

You can run the end-to-end integration tests using standard Go testing tools:
```bash
go test -v ./tests/...
```