package main

import (
	"log"
	"net/http"

	"github.com/c14220110/poliklinik-backend/config"
	adminControllers "github.com/c14220110/poliklinik-backend/internal/administrasi/controllers"
	adminRoutes "github.com/c14220110/poliklinik-backend/internal/administrasi/routes"
	adminServices "github.com/c14220110/poliklinik-backend/internal/administrasi/services"
	dokterControllers "github.com/c14220110/poliklinik-backend/internal/dokter/controllers"
	dokterRoutes "github.com/c14220110/poliklinik-backend/internal/dokter/routes"
	dokterServices "github.com/c14220110/poliklinik-backend/internal/dokter/services"
	screeningControllers "github.com/c14220110/poliklinik-backend/internal/screening/controllers"
	screeningRoutes "github.com/c14220110/poliklinik-backend/internal/screening/routes"
	screeningServices "github.com/c14220110/poliklinik-backend/internal/screening/services"
	"github.com/c14220110/poliklinik-backend/pkg/storage/mariadb"
)

func main() {
	// Muat konfigurasi dari .env
	cfg := config.LoadConfig()

	// Hubungkan ke database MariaDB
	db := mariadb.Connect()

	// Inisialisasi service untuk administrasi, pendaftaran, dan billing
	adminService := adminServices.NewAdministrasiService(db)
	pendaftaranService := adminServices.NewPendaftaranService(db)
	billingService := adminServices.NewBillingService(db)

	// Inisialisasi service untuk screening (suster)
	susterService := screeningServices.NewSusterService(db)

	// Inisialisasi service untuk dokter
	dokterService := dokterServices.NewDokterService(db)

	// Inisialisasi controller untuk administrasi
	adminController := adminControllers.NewAdministrasiController(adminService)
	pasienController := adminControllers.NewPasienController(pendaftaranService)
	billingController := adminControllers.NewBillingController(billingService)

	// Inisialisasi controller untuk screening (suster)
	susterController := screeningControllers.NewSusterController(susterService)

	// Inisialisasi controller untuk dokter
	dokterController := dokterControllers.NewDokterController(dokterService)

	// Daftarkan routing khusus untuk modul administrasi
	adminRoutes.RegisterAdministrasiRoutes(adminController, pasienController, billingController)

	// Daftarkan routing khusus untuk screening (suster)
	screeningRoutes.RegisterSusterRoutes(susterController)

	// Daftarkan routing khusus untuk dokter
	dokterRoutes.RegisterDokterRoutes(dokterController)

	log.Printf("Server berjalan pada port %s...", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}
