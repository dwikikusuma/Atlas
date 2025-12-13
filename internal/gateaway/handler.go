package gateaway

import (
	"encoding/json"
	"net/http"

	"github.com/dwikikusuma/atlas/pkg/pb/order"
	"github.com/dwikikusuma/atlas/pkg/pb/tracker"
)

type GatewayHandler struct {
	tracker tracker.TrackerServiceClient
	order   order.OrderServiceClient
}

func NewGatewayHandler(trackerClient tracker.TrackerServiceClient, orderClient order.OrderServiceClient) *GatewayHandler {
	return &GatewayHandler{
		tracker: trackerClient,
		order:   orderClient,
	}
}

// RegisterRoutes is a suggestion for Go 1.22+ routing
func (h *GatewayHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.HealthCheck)
	mux.HandleFunc("GET /order", h.GetOrder) // usage: /order?id=123
	mux.HandleFunc("POST /order", h.CreateOrder)
	mux.HandleFunc("PUT /order/status", h.UpdateOrderStatus)
}

// --- Handlers ---

func (h *GatewayHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{
		"status":  "ok",
		"service": "gateway",
	}
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "missing order id")
		return
	}

	orderDetail, err := h.order.GetOrder(r.Context(), &order.GetOrderRequest{OrderId: id})
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to get order: "+err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, orderDetail)
}

func (h *GatewayHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req order.CreateOrderRequest
	if !h.readJSON(w, r, &req) {
		return // readJSON handles the error response
	}

	orderResp, err := h.order.CreateOrder(r.Context(), &req)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to create order: "+err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, orderResp)
}

func (h *GatewayHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	var req order.UpdateOrderStatusRequest
	if !h.readJSON(w, r, &req) {
		return
	}

	orderResp, err := h.order.UpdateOrderStatus(r.Context(), &req)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to update order status: "+err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, orderResp)
}

func (h *GatewayHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *GatewayHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}

// readJSON decodes the body and handles the error if it fails.
// Returns true if successful, false if it failed (and response was already written).
func (h *GatewayHandler) readJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return false
	}
	return true
}
