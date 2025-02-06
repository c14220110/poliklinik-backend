package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/dokter/controllers"
)

func RegisterDokterRoutes(dc *controllers.DokterController) {
	http.HandleFunc("/api/dokter/register", dc.CreateDokter)
	http.HandleFunc("/api/dokter/login", dc.LoginDokter)
	// Endpoint ListAntrian dilindungi JWT
	http.Handle("/api/dokter/antrian", middlewares.JWTMiddleware(http.HandlerFunc(dc.ListAntrian)))
}
