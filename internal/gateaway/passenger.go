package gateaway

import (
	"net/http"

	"github.com/dwikikusuma/atlas/pkg/pb/dispatch"
	"github.com/dwikikusuma/atlas/pkg/pb/order"
)

type CustomerHandler struct {
	order    order.OrderServiceClient
	dispatch dispatch.DispatchServiceClient
}

func NewCustomerHandler(orderClient order.OrderServiceClient, dispatchClient dispatch.DispatchServiceClient) *CustomerHandler {
	return &CustomerHandler{
		order:    orderClient,
		dispatch: dispatchClient,
	}
}

func (h *CustomerHandler) RegisterRoutes(mux *http.ServeMux) {
	// Prefix: /customer
	mux.HandleFunc("POST /customer/order", h.CreateOrder)
	mux.HandleFunc("POST /customer/ride/request", h.RequestRide)
	mux.HandleFunc("GET /customer/order", h.GetOrder)
}

func (h *CustomerHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req order.CreateOrderRequest
	if !readJSON(w, r, &req) {
		return
	}

	resp, err := h.order.CreateOrder(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create order: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *CustomerHandler) RequestRide(w http.ResponseWriter, r *http.Request) {
	var req dispatch.RequestRideRequest
	if !readJSON(w, r, &req) {
		return
	}

	resp, err := h.dispatch.RequestRide(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to request ride: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *CustomerHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing order id")
		return
	}

	resp, err := h.order.GetOrder(r.Context(), &order.GetOrderRequest{OrderId: id})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get order: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
