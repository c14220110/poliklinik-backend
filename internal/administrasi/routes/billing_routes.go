package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/controllers"
	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
)

// RegisterBillingRoutes mendaftarkan endpoint billing yang dilindungi oleh JWT middleware Admin.
func RegisterBillingRoutes(bc *controllers.BillingController) {
	// Endpoint untuk mendapatkan billing terbaru (recent billing)
	http.Handle("/api/administrasi/billing/recent", middlewares.JWTMiddleware(http.HandlerFunc(bc.ListBilling)))
	
	// Endpoint untuk mendapatkan detail billing tertentu
	http.Handle("/api/administrasi/billing/detail", middlewares.JWTMiddleware(http.HandlerFunc(bc.BillingDetail)))
}
