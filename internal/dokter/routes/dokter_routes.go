package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/dokter/controllers"
)

func RegisterDokterRoutes(dc *controllers.DokterController) {
	http.HandleFunc("/api/dokter/login", dc.LoginDokter)
}
