package api

import (
	"encoding/json"
	"net/http"

	"github.com/a1234/notify-service/internal/model"
	"github.com/a1234/notify-service/internal/repository"
	"github.com/a1234/notify-service/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	notifService *service.NotificationService
	vendorRepo   repository.VendorConfigRepository
	pool         *pgxpool.Pool
}

func NewHandler(notifService *service.NotificationService, vendorRepo repository.VendorConfigRepository, pool *pgxpool.Pool) *Handler {
	return &Handler{
		notifService: notifService,
		vendorRepo:   vendorRepo,
		pool:         pool,
	}
}

func (h *Handler) CreateNotification(w http.ResponseWriter, r *http.Request) {
	var req model.CreateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.VendorID == uuid.Nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "vendor_id is required"})
		return
	}
	if req.Payload == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "payload is required"})
		return
	}

	n, err := h.notifService.Create(r.Context(), &req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, n)
}

func (h *Handler) GetNotification(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid notification id"})
		return
	}

	n, err := h.notifService.GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "notification not found"})
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (h *Handler) CreateVendor(w http.ResponseWriter, r *http.Request) {
	var req model.CreateVendorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" || req.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and url are required"})
		return
	}

	v, err := h.vendorRepo.Create(r.Context(), &req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (h *Handler) ListVendors(w http.ResponseWriter, r *http.Request) {
	vendors, err := h.vendorRepo.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if vendors == nil {
		vendors = []model.VendorConfig{}
	}
	writeJSON(w, http.StatusOK, vendors)
}

func (h *Handler) GetVendor(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid vendor id"})
		return
	}

	v, err := h.vendorRepo.GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "vendor not found"})
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (h *Handler) UpdateVendor(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid vendor id"})
		return
	}

	var req model.UpdateVendorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	v, err := h.vendorRepo.Update(r.Context(), id, &req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (h *Handler) DeleteVendor(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid vendor id"})
		return
	}

	if err := h.vendorRepo.Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if err := h.pool.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unhealthy", "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
