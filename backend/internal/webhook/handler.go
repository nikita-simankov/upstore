package webhook

import (
	"encoding/json"
	"net/http"

	"github.com/nikita-simankov/upstore/internal/order"
)

type Handler struct {
	orderRepo *order.Repository
}

func NewHandler(or *order.Repository) *Handler {
	return &Handler{orderRepo: or}
}

type yukassaEvent struct {
	Type   string `json:"type"`
	Object struct {
		ID     string            `json:"id"`
		Status string            `json:"status"`
		Meta   map[string]string `json:"metadata"`
	} `json:"object"`
}

func (h *Handler) Yukassa(w http.ResponseWriter, r *http.Request) {
	var event yukassaEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	switch event.Type {
	case "payment.succeeded":
		_ = h.orderRepo.UpdateStatus(r.Context(), event.Object.ID, "paid")
	case "payment.canceled":
		_ = h.orderRepo.UpdateStatus(r.Context(), event.Object.ID, "cancelled")
	}
	w.WriteHeader(http.StatusOK)
}
