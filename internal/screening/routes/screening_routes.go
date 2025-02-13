package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/screening/controllers"
)

func RegisterScreeningRoutes(sc *controllers.ScreeningController) {
	// Endpoint POST untuk input screening
	http.Handle("/api/screening/input", middlewares.JWTMiddleware(http.HandlerFunc(sc.InputScreening)))
	
	// Endpoint GET untuk mendapatkan screening berdasarkan id_pasien
	http.Handle("/api/screening", middlewares.JWTMiddleware(http.HandlerFunc(sc.GetScreeningByPasienHandler)))
}
