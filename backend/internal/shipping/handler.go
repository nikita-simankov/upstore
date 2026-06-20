package shipping

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type Handler struct {
	cdek *CDEKClient
}

func NewHandler(cdek *CDEKClient) *Handler {
	return &Handler{cdek: cdek}
}

func (h *Handler) Rates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.cdek == nil {
		json.NewEncoder(w).Encode([]ShippingRate{})
		return
	}
	fromCity, _ := strconv.Atoi(r.URL.Query().Get("from"))
	toCity, _ := strconv.Atoi(r.URL.Query().Get("to"))
	weight, _ := strconv.Atoi(r.URL.Query().Get("weight"))
	if fromCity == 0 {
		fromCity = 44
	}
	if weight == 0 {
		weight = 500
	}
	if toCity == 0 {
		json.NewEncoder(w).Encode([]ShippingRate{})
		return
	}
	rates, err := h.cdek.CalculateRates(r.Context(), fromCity, toCity, weight)
	if err != nil {
		json.NewEncoder(w).Encode([]ShippingRate{})
		return
	}
	json.NewEncoder(w).Encode(rates)
}
