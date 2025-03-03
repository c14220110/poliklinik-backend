package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/c14220110/poliklinik-backend/config"

	"github.com/c14220110/poliklinik-backend/internal/routes"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	sembarang "github.com/labstack/echo/v4/middleware"

	"github.com/c14220110/poliklinik-backend/pkg/storage/mariadb"
)

func main() {
	// Load file .env
	if err := godotenv.Load(".env"); err != nil {
		slog.Error("Error loading app.env file", "reason", err)
		os.Exit(1)
	}

	// Load konfigurasi
	cfg := config.LoadConfig()

	// Inisialisasi koneksi database
	db := mariadb.Connect()

	// Inisialisasi Echo
	e := echo.New()

	// Inisialisasi logger terstruktur dengan slog
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	// Terapkan middleware global (misalnya CORS)
	e.Use(sembarang.CORSWithConfig(sembarang.DefaultCORSConfig))

	// Daftarkan semua routes
	routes.Init(e, db)

	// Jalankan server di goroutine
	go func() {
		slog.Info("Starting server", "port", cfg.Port)
		if err := e.Start(":" + cfg.Port); err != nil && err != http.ErrServerClosed {
			slog.Error("Shutting down the server", "reason", err)
			os.Exit(1)
		}
	}()

	// Tangani graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	slog.Info("Received shutdown signal. Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		slog.Error("Error during server shutdown", "reason", err)
	}

	// Tutup koneksi database
	if err := mariadb.Close(); err != nil {
		slog.Error("Error closing database connection", "reason", err)
	}

	slog.Info("Server gracefully stopped")
}