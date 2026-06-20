package checkout

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nikita-simankov/upstore/internal/order"
	"github.com/nikita-simankov/upstore/internal/payment"
	"github.com/nikita-simankov/upstore/internal/product"
	storeRepo "github.com/nikita-simankov/upstore/internal/store"
)

type Handler struct {
	storeRepo   *storeRepo.Repository
	productRepo *product.Repository
	orderRepo   *order.Repository
	yukassa     *payment.YukassaClient
	baseURL     string
}

func NewHandler(sr *storeRepo.Repository, pr *product.Repository, or *order.Repository, yk *payment.YukassaClient, baseURL string) *Handler {
	return &Handler{storeRepo: sr, productRepo: pr, orderRepo: or, yukassa: yk, baseURL: baseURL}
}

type checkoutRequest struct {
	CustomerName    string  `json:"customer_name"`
	CustomerEmail   string  `json:"customer_email"`
	CustomerPhone   *string `json:"customer_phone"`
	ShippingAddress string  `json:"shipping_address"`
	PaymentProvider string  `json:"payment_provider"`
	Items           []struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	} `json:"items"`
}

type checkoutResponse struct {
	OrderID    string `json:"order_id"`
	PaymentURL string `json:"payment_url,omitempty"`
	Total      int    `json:"total"`
}

func (h *Handler) Checkout(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	s, err := h.storeRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, "store not found", http.StatusNotFound)
		return
	}

	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.CustomerName == "" || req.CustomerEmail == "" || req.ShippingAddress == "" || len(req.Items) == 0 {
		http.Error(w, "customer_name, customer_email, shipping_address and items are required", http.StatusBadRequest)
		return
	}
	if req.PaymentProvider == "" {
		req.PaymentProvider = "yukassa"
	}

	var items []order.Item
	total := 0
	for _, ri := range req.Items {
		if ri.Quantity <= 0 {
			http.Error(w, "invalid quantity", http.StatusBadRequest)
			return
		}
		p, err := h.productRepo.GetByID(r.Context(), ri.ProductID, s.ID)
		if err != nil || !p.Published {
			http.Error(w, fmt.Sprintf("product %s not available", ri.ProductID), http.StatusBadRequest)
			return
		}
		items = append(items, order.Item{
			ProductID: p.ID,
			Name:      p.Name,
			Price:     p.Price,
			Quantity:  ri.Quantity,
		})
		total += p.Price * ri.Quantity
	}

	o, err := h.orderRepo.Create(r.Context(), order.CreateOrderInput{
		StoreID:         s.ID,
		CustomerEmail:   req.CustomerEmail,
		CustomerName:    req.CustomerName,
		CustomerPhone:   req.CustomerPhone,
		ShippingAddress: req.ShippingAddress,
		Total:           total,
		PaymentProvider: req.PaymentProvider,
		Items:           items,
	})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if h.yukassa == nil {
		json.NewEncoder(w).Encode(checkoutResponse{OrderID: o.ID, Total: total})
		return
	}

	returnURL := fmt.Sprintf("%s/%s/order/%s", h.baseURL, slug, o.ID)
	pay, err := h.yukassa.CreatePayment(r.Context(), total, o.ID, returnURL, o.ID)
	if err != nil {
		json.NewEncoder(w).Encode(checkoutResponse{OrderID: o.ID, Total: total})
		return
	}

	_ = h.orderRepo.UpdatePaymentID(r.Context(), o.ID, pay.ID)
	json.NewEncoder(w).Encode(checkoutResponse{OrderID: o.ID, PaymentURL: pay.Confirmation.ConfirmationURL, Total: total})
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "orderId")
	o, err := h.orderRepo.GetByID(r.Context(), orderID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(o)
}
