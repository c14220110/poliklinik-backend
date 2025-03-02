package routes

import (
	"database/sql"

	"github.com/labstack/echo/v4"

	adminControllers "github.com/c14220110/poliklinik-backend/internal/administrasi/controllers"
	adminServices "github.com/c14220110/poliklinik-backend/internal/administrasi/services"
	manajemenControllers "github.com/c14220110/poliklinik-backend/internal/manajemen/controllers"
	manajemenServices "github.com/c14220110/poliklinik-backend/internal/manajemen/services"
	screeningControllers "github.com/c14220110/poliklinik-backend/internal/screening/controllers"
	screeningServices "github.com/c14220110/poliklinik-backend/internal/screening/services"

	// Asumsi package dokter sudah ada
	//dokterControllers "github.com/c14220110/poliklinik-backend/internal/dokter/controllers"
	//dokterServices "github.com/c14220110/poliklinik-backend/internal/dokter/services"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
)

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

	//dokterService := dokterServices.NewDokterService(db) // Asumsi

	privilegeService := manajemenServices.NewPrivilegeService(db)


	// Inisialisasi controller
	adminController := adminControllers.NewAdministrasiController(adminService)
	pasienController := adminControllers.NewPasienController(pendaftaranService)
	billingController := adminControllers.NewBillingController(billingService)
	poliklinikController := adminControllers.NewPoliklinikController(poliklinikService)

	managementController := manajemenControllers.NewManagementController(managementService)
	karyawanController := manajemenControllers.NewKaryawanController(managementService)
	roleController := manajemenControllers.NewRoleController(roleService)
	shiftController := manajemenControllers.NewShiftController(shiftService)
	cmsController := manajemenControllers.NewCMSController(cmsService)

	susterController := screeningControllers.NewSusterController(susterService)
	screeningController := screeningControllers.NewScreeningController(screeningService)
	antrianController := screeningControllers.NewAntrianController(antrianService)

	privilegeController := manajemenControllers.NewPrivilegeController(privilegeService)


	//dokterController := dokterControllers.NewDokterController(dokterService)

	// Grup API utama
	api := e.Group("/api")

	// 1. Administrasi (Aplikasi Pendaftaran & Administrasi)
	administrasi := api.Group("/administrasi")
	administrasi.POST("/login", adminController.Login) // Tidak pakai JWT
	administrasi.GET("/polikliniklist", poliklinikController.GetPoliklinikList, middlewares.JWTMiddleware())
	administrasi.GET("/pasiendata", pasienController.GetAllPasienData, middlewares.JWTMiddleware())
	administrasi.GET("/pasien", pasienController.ListPasien, middlewares.JWTMiddleware())

	// Kunjungan & antrian
	administrasi.PUT("/kunjungan", pasienController.UpdateKunjungan, middlewares.JWTMiddleware())
	administrasi.PUT("/kunjungan/reschedule", pasienController.RescheduleAntrianHandler, middlewares.JWTMiddleware())
	administrasi.PUT("/kunjungan/tunda", pasienController.TundaPasienHandler, middlewares.JWTMiddleware())
	administrasi.GET("/antrian/today", pasienController.GetAntrianTodayHandler, middlewares.JWTMiddleware())
	administrasi.GET("/status_antrian", pasienController.GetAllStatusAntrianHandler, middlewares.JWTMiddleware())

	// Billing
	billing := administrasi.Group("/billing")
	billing.GET("/recent", billingController.ListBilling, middlewares.JWTMiddleware())
	billing.GET("/detail", billingController.BillingDetail, middlewares.JWTMiddleware())

	// 2. Screening / Suster (Aplikasi Screening)
	screening := api.Group("/screening")
	screening.POST("/suster/login", susterController.LoginSuster) // Tidak pakai JWT
	screening.POST("/input", screeningController.InputScreening, middlewares.JWTMiddleware())
	screening.GET("", screeningController.GetScreeningByPasienHandler, middlewares.JWTMiddleware())
	screening.GET("/antrian/terlama", antrianController.GetAntrianTerlamaHandler) // Tidak pakai JWT
	screening.POST("/masukkan", antrianController.MasukkanPasienHandler, middlewares.JWTMiddleware())

	// 3. Dokter (Website untuk Dokter)
	// dokter := api.Group("/dokter")
	// dokter.POST("/login", dokterController.Login, middlewares.JWTMiddleware())         // Dokter login
	// dokter.GET("/antrian", dokterController.GetAntrianList, middlewares.JWTMiddleware())  // List antrian
	// dokter.GET("/detail", dokterController.GetPasienDetail, middlewares.JWTMiddleware())  // Rincian data pasien
	// dokter.POST("/assessment", dokterController.AddAssessment, middlewares.JWTMiddleware()) // Tambah assessment
	// dokter.POST("/e-resep", dokterController.AddEresep, middlewares.JWTMiddleware())      // Tambah e-resep
	// dokter.PUT("/pulangkan", dokterController.PulangkanPasien, middlewares.JWTMiddleware()) // Pulangkan pasien
	// Tambahkan endpoint dokter lain sesuai kebutuhan

	// 4. Management (Website untuk Manajemen)
	management := api.Group("/management")
	management.POST("/login", managementController.Login) // Tidak pakai JWT

	// Manajemen Karyawan
	management.POST("/karyawan", karyawanController.AddKaryawan, middlewares.JWTMiddleware())
	management.GET("/karyawan", karyawanController.GetKaryawanListHandler, middlewares.JWTMiddleware())
	management.PUT("/karyawan/update", karyawanController.UpdateKaryawanHandler, middlewares.JWTMiddleware())
	management.DELETE("/karyawan/delete", karyawanController.SoftDeleteKaryawanHandler, middlewares.JWTMiddleware())

	// Manajemen Role
	management.POST("/role/add", roleController.AddRoleHandler, middlewares.JWTMiddleware())
	management.PUT("/role/update", roleController.UpdateRoleHandler, middlewares.JWTMiddleware())
	management.PUT("/role/nonaktifkan", roleController.SoftDeleteRoleHandler, middlewares.JWTMiddleware())
	management.PUT("/role/aktifkan", roleController.ActivateRoleHandler, middlewares.JWTMiddleware())
	management.GET("/role/list", roleController.GetRoleListHandler, middlewares.JWTMiddleware())

	// Manajemen Privilege
	management.POST("/privilege/add", karyawanController.AddPrivilegeHandler, middlewares.JWTMiddleware())
	management.GET("/privilege", privilegeController.GetAllPrivilegesHandler, middlewares.JWTMiddleware())



	// Manajemen Shift & CMS
	management.POST("/shift/assign", shiftController.AssignShiftHandler, middlewares.JWTMiddleware())
	management.PUT("/shift/updateCustom", shiftController.UpdateCustomShiftHandler, middlewares.JWTMiddleware())
	management.PUT("/shift/soft-delete", shiftController.SoftDeleteShiftHandler, middlewares.JWTMiddleware())

	management.GET("/cms", cmsController.GetCMSByPoliklinikHandler, middlewares.JWTMiddleware())
	management.GET("/cms/all", cmsController.GetAllCMSHandler, middlewares.JWTMiddleware())
	management.POST("/cms/create", cmsController.CreateCMSHandler, middlewares.JWTMiddleware())
	management.PUT("/cms/update", cmsController.UpdateCMSHandler, middlewares.JWTMiddleware())
}
