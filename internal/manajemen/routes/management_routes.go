package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/manajemen/controllers"
)

func RegisterManagementRoutes(mc *controllers.ManagementController) {
	// Login tidak dilindungi
	http.HandleFunc("/api/management/login", mc.Login)
	// Endpoint lainnya untuk dashboard dll. dilindungi oleh JWT middleware di dalam controller (atau di route).
}
