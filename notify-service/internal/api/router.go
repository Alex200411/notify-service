package api

import "net/http"

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/notifications", h.CreateNotification)
	mux.HandleFunc("GET /api/v1/notifications/{id}", h.GetNotification)

	mux.HandleFunc("POST /api/v1/vendors", h.CreateVendor)
	mux.HandleFunc("GET /api/v1/vendors", h.ListVendors)
	mux.HandleFunc("GET /api/v1/vendors/{id}", h.GetVendor)
	mux.HandleFunc("PUT /api/v1/vendors/{id}", h.UpdateVendor)
	mux.HandleFunc("DELETE /api/v1/vendors/{id}", h.DeleteVendor)

	mux.HandleFunc("GET /healthz", h.HealthCheck)

	var handler http.Handler = mux
	handler = logging(handler)
	handler = requestID(handler)
	handler = recovery(handler)

	return handler
}
