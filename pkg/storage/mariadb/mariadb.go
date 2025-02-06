package mariadb

import (
	"database/sql"
	"fmt"
	"log"
	"sync"

	"github.com/c14220110/poliklinik-backend/config"
	_ "github.com/go-sql-driver/mysql"
)

var (
	db   *sql.DB
	once sync.Once
)

// Connect membuka koneksi ke database MariaDB.
// Semua kredensial dan informasi sensitif diambil dari file .env melalui config.go.
func Connect() *sql.DB {
	once.Do(func() {
		cfg := config.LoadConfig()
		// Format DSN: username:password@tcp(host:port)/dbname?parseTime=true&loc=Asia%2FJakarta
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Asia%%2FJakarta",
			cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

		var err error
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			log.Fatalf("Gagal membuka koneksi ke database: %v", err)
		}

		if err = db.Ping(); err != nil {
			log.Fatalf("Gagal melakukan ping ke database: %v", err)
		}

		log.Println("Berhasil terhubung ke MariaDB.")
	})

	return db
}

// GetDB mengembalikan instance koneksi database yang sudah terbentuk.
func GetDB() *sql.DB {
	return db
}
