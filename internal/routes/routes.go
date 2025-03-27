package routes

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v4"

	adminControllers "github.com/c14220110/poliklinik-backend/internal/administrasi/controllers"
	adminServices "github.com/c14220110/poliklinik-backend/internal/administrasi/services"
	"github.com/c14220110/poliklinik-backend/ws"

	manajemenControllers "github.com/c14220110/poliklinik-backend/internal/manajemen/controllers"
	manajemenServices "github.com/c14220110/poliklinik-backend/internal/manajemen/services"

	screeningControllers "github.com/c14220110/poliklinik-backend/internal/screening/controllers"
	screeningServices "github.com/c14220110/poliklinik-backend/internal/screening/services"

	dokterControllers "github.com/c14220110/poliklinik-backend/internal/dokter/controllers"
	dokterServices "github.com/c14220110/poliklinik-backend/internal/dokter/services"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
)

func Init(e *echo.Echo, db *sql.DB) {
	// Inisialisasi service untuk masing-masing modul
	// Administrasi
	adminService := adminServices.NewAdministrasiService(db)
	pendaftaranService := adminServices.NewPendaftaranService(db)
	billingService := adminServices.NewBillingService(db)
	// Untuk poliklinik, gunakan service dari manajemen
	poliklinikService := manajemenServices.NewPoliklinikService(db)

	// Management
	managementService := manajemenServices.NewManagementService(db)
	roleService := manajemenServices.NewRoleService(db)
	shiftService := manajemenServices.NewShiftService(db)
	cmsService := manajemenServices.NewCMSService(db)
	privilegeService := manajemenServices.NewPrivilegeService(db)

	// Screening / Suster
	screeningService := screeningServices.NewScreeningService(db)
	antrianService := screeningServices.NewAntrianService(db)
	susterService := screeningServices.NewSusterService(db)

	// Dokter
	dokterService := dokterServices.NewDokterService(db)

	// Inisialisasi controller
	// Administrasi
	adminController := adminControllers.NewAdministrasiController(adminService)
	pasienController := adminControllers.NewPasienController(pendaftaranService)
	billingController := adminControllers.NewBillingController(billingService)
	// Management (poliklinik, karyawan, role, shift, CMS, privilege)
	managementController := manajemenControllers.NewManagementController(managementService)
	karyawanController := manajemenControllers.NewKaryawanController(managementService)
	roleController := manajemenControllers.NewRoleController(roleService)
	shiftController := manajemenControllers.NewShiftController(shiftService)
	cmsController := manajemenControllers.NewCMSController(cmsService)
	poliklinikController := manajemenControllers.NewPoliklinikController(poliklinikService)
	privilegeController := manajemenControllers.NewPrivilegeController(privilegeService)
	// Screening / Suster
	susterController := screeningControllers.NewSusterController(susterService)
	screeningController := screeningControllers.NewScreeningController(screeningService)
	antrianController := screeningControllers.NewAntrianController(antrianService)
	// Dokter
	dokterController := dokterControllers.NewDokterController(dokterService)

	// Grup API utama
	api := e.Group("/api")
	api.GET("/ws", ws.ServeWS(ws.HubInstance))
	api.GET("/ws-test", func(c echo.Context) error {
    ws.HubInstance.Broadcast <- []byte("Test broadcast message")
    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "Broadcast message sent",
    })
})




	// 1. Administrasi (Aplikasi Pendaftaran & Administrasi)
	administrasi := api.Group("/administrasi")
	administrasi.POST("/login", adminController.Login) // Tidak pakai JWT
	administrasi.GET("/pasien", pasienController.GetAllPasienData, middlewares.JWTMiddleware())
	administrasi.POST("/pasien/register", pasienController.RegisterPasien, middlewares.JWTMiddleware())
	administrasi.PUT("/kunjungan", pasienController.UpdateKunjungan, middlewares.JWTMiddleware())
	administrasi.PUT("/antrian/reschedule", pasienController.RescheduleAntrianHandler, middlewares.JWTMiddleware())
	administrasi.PUT("/antrian/tunda", pasienController.TundaPasienHandler, middlewares.JWTMiddleware())
	administrasi.GET("/antrian/today", pasienController.GetAntrianTodayHandler, middlewares.JWTMiddleware())
	administrasi.GET("/status_antrian", pasienController.GetAllStatusAntrianHandler, middlewares.JWTMiddleware())
	administrasi.GET("/poliklinik", poliklinikController.GetPoliklinikList, middlewares.JWTMiddleware())
	administrasi.PUT("/antrian/batalkan", pasienController.BatalkanAntrianHandler, middlewares.JWTMiddleware())



	billing := administrasi.Group("/billing")
	billing.GET("", billingController.ListBilling, middlewares.JWTMiddleware())

	// 2. Screening / Suster (Aplikasi Screening)
	screening := api.Group("/screening")
	screening.POST("/suster/login", susterController.LoginSuster) // Tidak pakai JWT
	screening.POST("/input", screeningController.InputScreening, middlewares.JWTMiddleware())
	screening.GET("", screeningController.GetScreeningByPasienHandler, middlewares.JWTMiddleware())
	screening.GET("/antrian/terlama", antrianController.GetAntrianTerlamaHandler, middlewares.JWTMiddleware())
	screening.PUT("/masukkan", antrianController.MasukkanPasienHandler, middlewares.JWTMiddleware())
	screening.GET("/poliklinik", poliklinikController.GetActivePoliklinikList)
	screening.PUT("/alihkan-pasien", antrianController.AlihkanPasienHandler, middlewares.JWTMiddleware())

	// 3. Dokter (Website untuk Dokter)
	dokter := api.Group("/dokter")
	dokter.POST("/login", dokterController.LoginDokter) // Tidak pakai JWT
	dokter.GET("/poliklinik", poliklinikController.GetActivePoliklinikList)
	dokter.GET("/antrian/terlama", antrianController.GetAntrianTerlamaHandler, middlewares.JWTMiddleware())
	dokter.POST("/input-screening", screeningController.InputScreening, middlewares.JWTMiddleware())
	dokter.GET("/screening", screeningController.GetScreeningByPasienHandler, middlewares.JWTMiddleware())
	dokter.PUT("/masukkan", antrianController.MasukkanPasienKeDokterHandler, middlewares.JWTMiddleware())
	dokter.PUT("/pulangkan-pasien", antrianController.PulangkanPasienHandler, middlewares.JWTMiddleware())


	// Tambahkan endpoint dokter lain sesuai kebutuhan

	// 4. Management (Website untuk Manajemen)
	management := api.Group("/management")
	management.POST("/login", managementController.Login) // Tidak pakai JWT

	// Manajemen Karyawan
	management.POST("/karyawan", karyawanController.AddKaryawan, middlewares.JWTMiddleware())
	management.GET("/karyawan", karyawanController.GetKaryawanListHandler, middlewares.JWTMiddleware())
	management.PUT("/karyawan/update", karyawanController.UpdateKaryawanHandler, middlewares.JWTMiddleware())
	management.PUT("/karyawan/delete", karyawanController.SoftDeleteKaryawanHandler, middlewares.JWTMiddleware())
	management.POST("/karyawan/addRole", karyawanController.AddRoleHandler, middlewares.JWTMiddleware())

	// Manajemen Poliklinik
	management.GET("/poliklinik", poliklinikController.GetPoliklinikList, middlewares.JWTMiddleware())
	management.POST("/poliklinik/add", poliklinikController.AddPoliklinikHandler, middlewares.JWTMiddleware())
	management.PUT("/poliklinik/update", poliklinikController.UpdatePoliklinikHandler, middlewares.JWTMiddleware())
	management.PUT("/poliklinik/soft-delete", poliklinikController.SoftDeletePoliklinikHandler, middlewares.JWTMiddleware())


	// Manajemen Role
	management.POST("/role/add", roleController.AddRoleHandler, middlewares.JWTMiddleware())
	management.PUT("/role/update", roleController.UpdateRoleHandler, middlewares.JWTMiddleware())
	management.PUT("/role/nonaktifkan", roleController.SoftDeleteRoleHandler, middlewares.JWTMiddleware())
	management.PUT("/role/aktifkan", roleController.ActivateRoleHandler, middlewares.JWTMiddleware())
	management.GET("/role/list", roleController.GetRoleListHandler, middlewares.JWTMiddleware())

	// Manajemen Privilege
	management.POST("/privilege/assign", karyawanController.AddPrivilegeHandler, middlewares.JWTMiddleware())
	management.GET("/privilege", privilegeController.GetAllPrivilegesHandler, middlewares.JWTMiddleware())
	management.POST("/privilege", privilegeController.CreatePrivilegeHandler, middlewares.JWTMiddleware())

	// Manajemen Shift & CMS
	management.POST("/shift/assign", shiftController.AssignShiftHandler, middlewares.JWTMiddleware())
	management.PUT("/shift/updateCustom", shiftController.UpdateCustomShiftHandler, middlewares.JWTMiddleware())
	management.PUT("/shift/soft-delete", shiftController.SoftDeleteShiftHandler, middlewares.JWTMiddleware())
	management.GET("/shift", shiftController.GetShiftPoliList, middlewares.JWTMiddleware())
	management.GET("/cms", cmsController.GetCMSByPoliklinikHandler, middlewares.JWTMiddleware())
	management.GET("/cms/all", cmsController.GetAllCMSHandler, middlewares.JWTMiddleware())
	management.POST("/cms/create", cmsController.CreateCMSHandler, middlewares.JWTMiddleware())
	management.PUT("/cms/update", cmsController.UpdateCMSHandler, middlewares.JWTMiddleware())
	management.GET("/shift/karyawan", shiftController.GetKaryawanListHandler, middlewares.JWTMiddleware())
	management.GET("/karyawan/tanpa-shift", shiftController.GetKaryawanTanpaShiftHandler, middlewares.JWTMiddleware())
	management.POST("/shift/assign-new", shiftController.AssignShiftHandlerNew, middlewares.JWTMiddleware())

}
