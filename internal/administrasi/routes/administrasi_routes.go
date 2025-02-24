package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/controllers"
	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
)

func RegisterAdministrasiRoutes(ac *controllers.AdministrasiController, pc *controllers.PasienController, bc *controllers.BillingController) {
	// Login endpoint tidak dilindungi
	http.HandleFunc("/api/administrasi/login", ac.Login)

	// Registrasi pasien (POST)
	http.Handle("/api/pasien/register", middlewares.JWTMiddleware(http.HandlerFunc(pc.RegisterPasien)))

	//update
	http.Handle("/api/kunjungan", middlewares.JWTMiddleware(http.HandlerFunc(pc.UpdateKunjungan)))


	// Endpoint untuk mengambil seluruh data pasien
	http.Handle("/api/administrasi/pasiendata", middlewares.JWTMiddleware(http.HandlerFunc(pc.GetAllPasienData)))

	// Endpoint list pasien (bisa berbeda, sesuai kebutuhan)
	http.Handle("/api/administrasi/pasien", middlewares.JWTMiddleware(http.HandlerFunc(pc.ListPasien)))

	// Endpoint billing
	
		http.Handle("/api/kunjungan/reschedule", middlewares.JWTMiddleware(http.HandlerFunc(pc.RescheduleAntrianHandler)))

			http.Handle("/api/kunjungan/tunda", middlewares.JWTMiddleware(http.HandlerFunc(pc.TundaPasienHandler)))


}
