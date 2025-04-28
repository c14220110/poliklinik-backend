package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/manajemen/services"
	"github.com/labstack/echo/v4"
)

type DashboardController struct {
	Service *services.DashboardService
}

func NewDashboardController(svc *services.DashboardService) *DashboardController {
	return &DashboardController{Service: svc}
}

// GetDashboard handles GET /management/dashboard
func (dc *DashboardController) GetDashboard(c echo.Context) error {
	var idPoli *int
	if s := c.QueryParam("id_poli"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid id_poli"})
		}
		idPoli = &v
	}

	loc, _ := time.LoadLocation("Asia/Jakarta")
	// parse start date or default to today at 00:00
	now := time.Now().In(loc)
	parseStart := func(s string) (time.Time, error) {
		if s == "" {
			return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc), nil
		}
		return time.ParseInLocation("02/01/2006", s, loc)
	}
	// parse end date or default to today at 00:00 (service will extend to end of day)
	parseEnd := func(s string) (time.Time, error) {
		if s == "" {
			return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc), nil
		}
		return time.ParseInLocation("02/01/2006", s, loc)
	}

	start, err := parseStart(c.QueryParam("rentang_awal"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid rentang_awal"})
	}
	end, err := parseEnd(c.QueryParam("rentang_akhir"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid rentang_akhir"})
	}

	dash, err := dc.Service.GetDashboardData(idPoli, start, end)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "failed to get dashboard: " + err.Error()})
	}

	return c.JSON(http.StatusOK, dash)
}
