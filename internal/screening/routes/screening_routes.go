package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/screening/controllers"
)

func RegisterScreeningRoutes(sc *controllers.ScreeningController) {
	// Misalnya, endpoint untuk input screening, dilindungi JWT (role Suster)
	http.Handle("/api/screening", middlewares.JWTMiddlewareSuster(http.HandlerFunc(sc.InputScreening)))
}
