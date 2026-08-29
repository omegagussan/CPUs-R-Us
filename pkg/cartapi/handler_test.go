package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
