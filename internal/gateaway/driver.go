package gateaway

import (
	"net/http"

	"github.com/dwikikusuma/atlas/pkg/pb/order"
	"github.com/dwikikusuma/atlas/pkg/pb/tracker"
)

type DriverHandler struct {
	tracker tracker.TrackerServiceClient
	order   order.OrderServiceClient
}

func NewDriverHandler(trackerClient tracker.TrackerServiceClient, orderClient order.OrderServiceClient) *DriverHandler {
	return &DriverHandler{
		tracker: trackerClient,
		order:   orderClient,
	}
}

func (h *DriverHandler) RegisterRoutes(mux *http.ServeMux) {
	// Prefix: /driver
	mux.HandleFunc("POST /driver/location", h.UpdateLocation)
	mux.HandleFunc("PUT /driver/order/status", h.UpdateOrderStatus)
}

func (h *DriverHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	var req tracker.UpdateLocationRequest
	if !readJSON(w, r, &req) {
		return
	}

	resp, err := h.tracker.UpdateLocation(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update location: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *DriverHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	var req order.UpdateOrderStatusRequest
	if !readJSON(w, r, &req) {
		return
	}

	// Drivers usually only trigger STARTED or FINISHED
	resp, err := h.order.UpdateOrderStatus(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update status: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
