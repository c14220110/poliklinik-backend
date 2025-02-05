package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/screening/controllers"
)

// RegisterAntrianRoutes menghubungkan endpoint untuk antrian screening.
func RegisterAntrianRoutes(ac *controllers.AntrianController) {
	http.HandleFunc("/api/screening/antrian/terlama", ac.GetAntrianTerlamaHandler)
}
