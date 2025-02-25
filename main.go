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
	manajemenControllers "github.com/c14220110/poliklinik-backend/internal/manajemen/controllers"
	manajemenRoutes "github.com/c14220110/poliklinik-backend/internal/manajemen/routes"
	manajemenServices "github.com/c14220110/poliklinik-backend/internal/manajemen/services"
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

    // Inisialisasi service untuk manajemen dan shift
    managementService := manajemenServices.NewManagementService(db)
    shiftService := manajemenServices.NewShiftService(db)
    cmsService := manajemenServices.NewCMSService(db)

    //Inisialisasi service untuk manajemen Role
    roleService := manajemenServices.NewRoleService(db)

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

    // Inisialisasi controller untuk manajemen
    managementController := manajemenControllers.NewManagementController(managementService)
    karyawanController := manajemenControllers.NewKaryawanController(managementService)

    // Inisialisasi controller untuk shift management (bagian dari Website Manajemen)
    shiftController := manajemenControllers.NewShiftController(shiftService, db)
    cmsController := manajemenControllers.NewCMSController(cmsService)

    // Manajemen Role
    roleController := manajemenControllers.NewRoleController(roleService)

    // Daftarkan routing untuk masing-masing modul
    adminRoutes.RegisterAdministrasiRoutes(adminController, pasienController, billingController)
    adminRoutes.RegisterPoliklinikRoutes(poliklinikController)
    screeningRoutes.RegisterSusterRoutes(susterController)
    screeningRoutes.RegisterScreeningRoutes(screeningController)
    dokterRoutes.RegisterDokterRoutes(dokterController)
    manajemenRoutes.RegisterManagementLoginRoutes(managementController)
    manajemenRoutes.RegisterKaryawanRoutes(karyawanController)
    manajemenRoutes.RegisterShiftRoutes(shiftController)
    manajemenRoutes.RegisterCMSRoutes(cmsController)
    manajemenRoutes.RegisterRoleRoutes(roleController)

    // Pastikan server berjalan dengan baik
    log.Printf("Server berjalan pada port %s...", cfg.Port)
    log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}