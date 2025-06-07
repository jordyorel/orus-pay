package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// LoadEnv loads variables from a .env file if present.
func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Printf("no .env file found: %v", err)
	}
}

// GetEnv returns an environment variable or a default value.
func GetEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

// GetIntEnv returns an int environment variable or a default value.
func GetIntEnv(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

// IsProduction checks if the app runs in production mode.
func IsProduction() bool {
	return GetEnv("ENV", "development") == "production"
}
