package main

import (
	"log"
	"net/http"

	"github.com/c14220110/poliklinik-backend/config"
	"github.com/c14220110/poliklinik-backend/internal/administrasi/controllers"
	"github.com/c14220110/poliklinik-backend/internal/administrasi/routes"
	"github.com/c14220110/poliklinik-backend/internal/administrasi/services"
	"github.com/c14220110/poliklinik-backend/pkg/storage/mariadb"
)

func main() {
	// Muat konfigurasi dari .env
	cfg := config.LoadConfig()

	// Hubungkan ke database MariaDB
	db := mariadb.Connect()

	// Inisialisasi service untuk administrasi, pendaftaran, dan billing
	adminService := services.NewAdministrasiService(db)
	pendaftaranService := services.NewPendaftaranService(db)
	billingService := services.NewBillingService(db)

	// Inisialisasi controller
	adminController := controllers.NewAdministrasiController(adminService)
	pasienController := controllers.NewPasienController(pendaftaranService)
	billingController := controllers.NewBillingController(billingService)

	// Daftarkan routing khusus untuk modul administrasi
	routes.RegisterAdministrasiRoutes(adminController, pasienController, billingController)

	log.Printf("Server berjalan pada port %s...", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}
