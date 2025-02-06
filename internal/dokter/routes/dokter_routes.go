package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/dokter/controllers"
)

func RegisterDokterRoutes(dc *controllers.DokterController) {
	http.HandleFunc("/api/dokter/register", dc.CreateDokter)
	http.HandleFunc("/api/dokter/login", dc.LoginDokter)
	http.HandleFunc("/api/dokter/antrian", dc.ListAntrian)
}
