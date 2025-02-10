package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/screening/controllers"
)

func RegisterSusterRoutes(sc *controllers.SusterController) {
	http.HandleFunc("/api/screening/suster/login", sc.LoginSuster)
}
