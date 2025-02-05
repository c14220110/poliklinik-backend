package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/screening/controllers"
)

// RegisterSusterRoutes mengaitkan endpoint untuk suster.
//username= susowen, pass= susowen
func RegisterSusterRoutes(sc *controllers.SusterController) {
	http.HandleFunc("/api/screening/suster/register", sc.CreateSuster)
	http.HandleFunc("/api/screening/suster/login", sc.LoginSuster)
}
