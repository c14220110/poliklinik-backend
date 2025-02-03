package config

import (
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv    string
	Port      string
	DBUser    string
	DBPassword string
	DBHost    string
	DBPort    string
	DBName    string
	JWTSecret string // <-- Tambahkan variabel ini
}

var (
	cfg  *Config
	once sync.Once
)

func LoadConfig() *Config {
	once.Do(func() {
		if err := godotenv.Load(); err != nil {
			log.Println("Warning: .env file not found. Relying on environment variables.")
		}
		cfg = &Config{
			AppEnv:     os.Getenv("APP_ENV"),
			Port:       os.Getenv("PORT"),
			DBUser:     os.Getenv("DB_USER"),
			DBPassword: os.Getenv("DB_PASSWORD"),
			DBHost:     os.Getenv("DB_HOST"),
			DBPort:     os.Getenv("DB_PORT"),
			DBName:     os.Getenv("DB_NAME"),
			JWTSecret:  os.Getenv("JWT_SECRET"), // Ambil JWT_SECRET dari .env
		}
	})
	return cfg
}
