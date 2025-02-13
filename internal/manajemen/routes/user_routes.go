package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/manajemen/controllers"
)

func RegisterUserRoutes(uc *controllers.UserController) {
	http.Handle("/api/management/karyawan", middlewares.JWTMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			uc.GetKaryawanList(w, r)
		case http.MethodPut:
			uc.UpdateKaryawan(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("Method not allowed"))
		}
	})))
}
