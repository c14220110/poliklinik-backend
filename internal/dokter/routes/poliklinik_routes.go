package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/dokter/controllers"
)

func RegisterPoliklinikRoutes(pc *controllers.PoliklinikController) {
	http.HandleFunc("/api/dokter/poliklinik", pc.GetPoliklinikList)
}
