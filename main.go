package main

import (
	"log"
	"net/http"

	"github.com/c14220110/poliklinik-backend/config"
	adminControllers "github.com/c14220110/poliklinik-backend/internal/administrasi/controllers"
	adminRoutes "github.com/c14220110/poliklinik-backend/internal/administrasi/routes"
	billingRoutes "github.com/c14220110/poliklinik-backend/internal/administrasi/routes"
	adminServices "github.com/c14220110/poliklinik-backend/internal/administrasi/services"
	dokterControllers "github.com/c14220110/poliklinik-backend/internal/dokter/controllers"
	dokterRoutes "github.com/c14220110/poliklinik-backend/internal/dokter/routes"
	dokterServices "github.com/c14220110/poliklinik-backend/internal/dokter/services"
	managementControllers "github.com/c14220110/poliklinik-backend/internal/manajemen/controllers"
	shiftControllers "github.com/c14220110/poliklinik-backend/internal/manajemen/controllers"
	cmsRoutes "github.com/c14220110/poliklinik-backend/internal/manajemen/routes"
	managementRoutes "github.com/c14220110/poliklinik-backend/internal/manajemen/routes"
	shiftRoutes "github.com/c14220110/poliklinik-backend/internal/manajemen/routes"
	managementServices "github.com/c14220110/poliklinik-backend/internal/manajemen/services"
	shiftServices "github.com/c14220110/poliklinik-backend/internal/manajemen/services"
	screeningControllers "github.com/c14220110/poliklinik-backend/internal/screening/controllers"
	screeningRoutes "github.com/c14220110/poliklinik-backend/internal/screening/routes"
	screeningServices "github.com/c14220110/poliklinik-backend/internal/screening/services"

	"github.com/joho/godotenv"

	"github.com/c14220110/poliklinik-backend/pkg/storage/mariadb"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	cfg := config.LoadConfig()
	db := mariadb.Connect()

	// Inisialisasi service untuk administrasi, pendaftaran, dan billing
	adminService := adminServices.NewAdministrasiService(db)
	pendaftaranService := adminServices.NewPendaftaranService(db)
	billingService := adminServices.NewBillingService(db)

	// Inisialisasi service untuk screening (suster)
	susterService := screeningServices.NewSusterService(db)
	screeningService := screeningServices.NewScreeningService(db)

	// Inisialisasi service untuk dokter (dari tabel Karyawan)
	dokterService := dokterServices.NewDokterService(db)

	// Inisialisasi service untuk management dan shift
	managementService := managementServices.NewManagementService(db) // ManagementService yang digunakan
	shiftService := shiftServices.NewShiftService(db)
	cmsService := managementServices.NewCMSService(db)


	// Inisialisasi controller untuk administrasi
	adminController := adminControllers.NewAdministrasiController(adminService)
	pasienController := adminControllers.NewPasienController(pendaftaranService)
	billingController := adminControllers.NewBillingController(billingService)
	poliklinikController := adminControllers.NewPoliklinikController(db)

	// Inisialisasi controller untuk screening (suster)
	susterController := screeningControllers.NewSusterController(susterService)
	screeningController := screeningControllers.NewScreeningController(screeningService)

	// Inisialisasi controller untuk dokter
	dokterController := dokterControllers.NewDokterController(dokterService)

	// Inisialisasi controller untuk management
	managementController := managementControllers.NewManagementController(managementService)
	karyawanController := managementControllers.NewKaryawanController(managementService) // Berikan managementService pada controller

	// Inisialisasi controller untuk shift management (bagian dari Website Manajemen)
	shiftController := shiftControllers.NewShiftController(shiftService, db)

	cmsController := managementControllers.NewCMSController(cmsService)


	// Daftarkan routing untuk masing-masing modul
	adminRoutes.RegisterAdministrasiRoutes(adminController, pasienController, billingController)
	adminRoutes.RegisterPoliklinikRoutes(poliklinikController)
	screeningRoutes.RegisterSusterRoutes(susterController)
	screeningRoutes.RegisterScreeningRoutes(screeningController)
	dokterRoutes.RegisterDokterRoutes(dokterController)
	managementRoutes.RegisterManagementLoginRoutes(managementController)
	managementRoutes.RegisterKaryawanRoutes(karyawanController)
	
	shiftRoutes.RegisterShiftRoutes(shiftController)
	billingRoutes.RegisterBillingRoutes(billingController)
	cmsRoutes.RegisterCMSRoutes(cmsController)


	// Pastikan server berjalan dengan baik
	log.Printf("Server berjalan pada port %s...", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}
