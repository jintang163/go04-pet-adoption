package server

import (
	"net/http"

	"go04-pet-adoption/internal/handler"
)

func NewMux(h *handler.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	h.Routes(mux)
	return mux
}
