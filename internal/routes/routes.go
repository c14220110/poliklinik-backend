package routes

import (
	"database/sql"

	"github.com/labstack/echo/v4"

	adminControllers "github.com/c14220110/poliklinik-backend/internal/administrasi/controllers"
	adminServices "github.com/c14220110/poliklinik-backend/internal/administrasi/services"
	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	manajemenControllers "github.com/c14220110/poliklinik-backend/internal/manajemen/controllers"
	manajemenServices "github.com/c14220110/poliklinik-backend/internal/manajemen/services"
	screeningControllers "github.com/c14220110/poliklinik-backend/internal/screening/controllers"
	screeningServices "github.com/c14220110/poliklinik-backend/internal/screening/services"
)

// Init menginisialisasi semua routes menggunakan Echo framework
func Init(e *echo.Echo, db *sql.DB) {
    // Inisialisasi service
    adminService := adminServices.NewAdministrasiService(db)
    pendaftaranService := adminServices.NewPendaftaranService(db)
    billingService := adminServices.NewBillingService(db)
    poliklinikService := adminServices.NewPoliklinikService(db)
    managementService := manajemenServices.NewManagementService(db)
    shiftService := manajemenServices.NewShiftService(db)
    cmsService := manajemenServices.NewCMSService(db)
    roleService := manajemenServices.NewRoleService(db)
    screeningService := screeningServices.NewScreeningService(db)
    antrianService := screeningServices.NewAntrianService(db)
    susterService := screeningServices.NewSusterService(db)

    // Inisialisasi controller dengan service yang sesuai
    adminController := adminControllers.NewAdministrasiController(adminService)
    pasienController := adminControllers.NewPasienController(pendaftaranService)
    billingController := adminControllers.NewBillingController(billingService)
    poliklinikController := adminControllers.NewPoliklinikController(poliklinikService) // Diperbaiki: gunakan poliklinikService
    managementController := manajemenControllers.NewManagementController(managementService)
    karyawanController := manajemenControllers.NewKaryawanController(managementService)
    roleController := manajemenControllers.NewRoleController(roleService)
    shiftController := manajemenControllers.NewShiftController(shiftService)
    cmsController := manajemenControllers.NewCMSController(cmsService)
    susterController := screeningControllers.NewSusterController(susterService)
    screeningController := screeningControllers.NewScreeningController(screeningService)
    antrianController := screeningControllers.NewAntrianController(antrianService)

    // Grup API utama
    api := e.Group("/api")

    // **Grup Administrasi**
    administrasi := api.Group("/administrasi")
    administrasi.POST("/login", adminController.Login) // Tidak pakai JWT
    administrasi.GET("/polikliniklist", poliklinikController.GetPoliklinikList, middlewares.JWTMiddleware())
    administrasi.GET("/pasiendata", pasienController.GetAllPasienData, middlewares.JWTMiddleware())
    administrasi.GET("/pasien", pasienController.ListPasien, middlewares.JWTMiddleware())

    // Endpoint kunjungan
    administrasi.PUT("/kunjungan", pasienController.UpdateKunjungan, middlewares.JWTMiddleware())
    administrasi.PUT("/kunjungan/reschedule", pasienController.RescheduleAntrianHandler, middlewares.JWTMiddleware())
    administrasi.PUT("/kunjungan/tunda", pasienController.TundaPasienHandler, middlewares.JWTMiddleware())
    administrasi.GET("/antrian/today", pasienController.GetAntrianTodayHandler, middlewares.JWTMiddleware())
    administrasi.GET("/status_antrian", pasienController.GetAllStatusAntrianHandler, middlewares.JWTMiddleware())

    // Billing di bawah administrasi
    billing := administrasi.Group("/billing")
    billing.GET("/recent", billingController.ListBilling, middlewares.JWTMiddleware())
    billing.GET("/detail", billingController.BillingDetail, middlewares.JWTMiddleware())

    // **Grup Pasien**
    pasien := api.Group("/pasien")
    pasien.POST("/register", pasienController.RegisterPasien, middlewares.JWTMiddleware())

    // **Grup Management**
    management := api.Group("/management")
    management.POST("/login", managementController.Login) // Tidak pakai JWT
    api.Group("/shift").POST("/assign", shiftController.AssignShiftHandler, middlewares.JWTMiddleware())
    api.Group("/shift").PUT("/updateCustom", shiftController.UpdateCustomShiftHandler, middlewares.JWTMiddleware())
    api.Group("/shift").PUT("/soft-delete", shiftController.SoftDeleteShiftHandler, middlewares.JWTMiddleware())



    // **Grup Karyawan**
    karyawan := api.Group("/karyawan")
    karyawan.POST("", karyawanController.AddKaryawan, middlewares.JWTMiddleware())
    karyawan.GET("", karyawanController.GetKaryawanListHandler, middlewares.JWTMiddleware())
    karyawan.PUT("/update", karyawanController.UpdateKaryawanHandler, middlewares.JWTMiddleware())
    karyawan.DELETE("/delete", karyawanController.SoftDeleteKaryawanHandler, middlewares.JWTMiddleware())

    // **Grup Role**
    role := api.Group("/role")
    role.POST("/add", roleController.AddRoleHandler, middlewares.JWTMiddleware())
    role.PUT("/update", roleController.UpdateRoleHandler, middlewares.JWTMiddleware())
    role.PUT("/nonaktifkan", roleController.SoftDeleteRoleHandler, middlewares.JWTMiddleware())
    role.PUT("/aktifkan", roleController.ActivateRoleHandler, middlewares.JWTMiddleware())
    role.GET("/list", roleController.GetRoleListHandler, middlewares.JWTMiddleware())

    // **Grup Screening**
    screening := api.Group("/screening")
    screening.POST("/suster/login", susterController.LoginSuster) // Tidak pakai JWT
    screening.POST("/input", screeningController.InputScreening, middlewares.JWTMiddleware())
    screening.GET("", screeningController.GetScreeningByPasienHandler, middlewares.JWTMiddleware())
    screening.GET("/antrian/terlama", antrianController.GetAntrianTerlamaHandler) // Tidak pakai JWT
    screening.POST("/masukkan", antrianController.MasukkanPasienHandler, middlewares.JWTMiddleware())


    // **Grup CMS**
    cms := api.Group("/cms")
    cms.GET("", cmsController.GetCMSByPoliklinikHandler, middlewares.JWTMiddleware())
    cms.GET("/all", cmsController.GetAllCMSHandler, middlewares.JWTMiddleware())
    cms.POST("/create", cmsController.CreateCMSHandler, middlewares.JWTMiddleware())
    cms.PUT("/update", cmsController.UpdateCMSHandler, middlewares.JWTMiddleware())
}