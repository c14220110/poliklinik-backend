package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/manajemen/controllers"
)

func RegisterShiftRoutes(sc *controllers.ShiftController) {
	// Misalnya, endpoint untuk melihat daftar poliklinik (jika diperlukan)
	http.Handle("/api/management/poliklinik", middlewares.JWTMiddlewareManagement(http.HandlerFunc(sc.GetPoliklinikListHandler)))
	
	http.Handle("/api/management/shift_karyawan/karyawan", middlewares.JWTMiddlewareManagement(http.HandlerFunc(sc.GetKaryawanByShiftAndPoliHandler)))

	http.Handle("/api/management/shift_summary", middlewares.JWTMiddlewareManagement(http.HandlerFunc(sc.GetShiftSummaryHandler)))


}
