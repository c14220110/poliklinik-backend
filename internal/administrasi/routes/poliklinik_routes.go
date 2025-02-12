package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/controllers"
	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
)

func RegisterPoliklinikRoutes(pc *controllers.PoliklinikController) {
	http.Handle("/api/administrasi/polikliniklist", middlewares.JWTMiddlewareAdmin(http.HandlerFunc(pc.GetPoliklinikList)))
}
