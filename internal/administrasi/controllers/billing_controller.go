package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/services"
)

type BillingController struct {
	Service *services.BillingService
}

func NewBillingController(service *services.BillingService) *BillingController {
	return &BillingController{Service: service}
}

func (bc *BillingController) ListBilling(w http.ResponseWriter, r *http.Request) {
	statusParam := r.URL.Query().Get("status")
	var filterStatus *int = nil
	if statusParam != "" {
		if val, err := strconv.Atoi(statusParam); err == nil {
			filterStatus = &val
		}
	}

	data, err := bc.Service.GetRecentBilling(filterStatus)
	if err != nil {
		http.Error(w, "Failed to retrieve billing data", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (bc *BillingController) BillingDetail(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id_pasien")
	if idParam == "" {
		http.Error(w, "id_pasien is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, "Invalid id_pasien", http.StatusBadRequest)
		return
	}

	detail, err := bc.Service.GetBillingDetail(id)
	if err != nil {
		http.Error(w, "Failed to retrieve billing detail", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}
