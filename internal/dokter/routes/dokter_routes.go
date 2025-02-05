package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/dokter/controllers"
)

// RegisterDokterRoutes menghubungkan endpoint untuk modul dokter.
func RegisterDokterRoutes(dc *controllers.DokterController) {
	http.HandleFunc("/api/dokter/register", dc.CreateDokter)
	http.HandleFunc("/api/dokter/login", dc.LoginDokter)
}
