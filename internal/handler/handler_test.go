package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go04-pet-adoption/internal/auth"
	"go04-pet-adoption/internal/handler"
	"go04-pet-adoption/internal/server"
	"go04-pet-adoption/internal/service"
	"go04-pet-adoption/internal/store"
)

func TestHealthzAndLogin(t *testing.T) {
	mem := store.NewMemoryStore(time.Now, nil)
	hasher := auth.NewPasswordHasher()
	sessions := auth.NewSessionManager(time.Hour)
	svc := service.NewServices(mem, hasher, sessions, nil, 3)
	if err := store.SeedAdmin(nil, mem, hasher, "admin", "admin123"); err != nil {
		t.Fatal(err)
	}
	h := handler.New(svc, mem, sessions, nil)
	mux := server.NewMux(h)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("healthz %d", rec.Code)
	}

	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "admin123"})
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("login %d %s", rec.Code, rec.Body.String())
	}
}
