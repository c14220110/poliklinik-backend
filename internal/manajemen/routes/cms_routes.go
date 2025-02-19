package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/manajemen/controllers"
)

func RegisterCMSRoutes(cc *controllers.CMSController) {
	// Endpoint GET untuk mendapatkan CMS berdasarkan poliklinik (parameter: poliklinik_id)
	http.Handle("/api/cms", middlewares.JWTMiddleware(http.HandlerFunc(cc.GetCMSByPoliklinikHandler)))
	
	// Endpoint GET untuk mendapatkan semua CMS dikelompokkan berdasarkan poliklinik
	http.Handle("/api/cms/all", middlewares.JWTMiddleware(http.HandlerFunc(cc.GetAllCMSHandler)))

	http.Handle("/api/cms/create", middlewares.JWTMiddleware(http.HandlerFunc(cc.CreateCMSHandler)))

	http.Handle("/api/cms/update", middlewares.JWTMiddleware(http.HandlerFunc(cc.UpdateCMSHandler)))


}
