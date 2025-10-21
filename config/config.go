package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Port                   int    `json:"port"`
	Environment            string `json:"environment"`
	ReadTimeout            int    `json:"read_timeout"`
	WriteTimeout           int    `json:"write_timeout"`
	IdleTimeout            int    `json:"idle_timeout"`
	GCPProjectID           string `json:"gcp_project_id"`
	LogPath                string `json:"log_path"`
	LanguageDictionaryPath string `json:"language_dictionary_path"`
	GeminiAPIKey           string `json:"gemini_api_key"`
	AudioDBPath            string `json:"audio_db_path"`
}

// Load loads configuration from environment variables and .env file
func Load() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	config := &Config{
		Port:                   getEnvAsInt("PORT", 8080),
		Environment:            getEnv("ENVIRONMENT", "development"),
		ReadTimeout:            getEnvAsInt("READ_TIMEOUT", 10),
		WriteTimeout:           getEnvAsInt("WRITE_TIMEOUT", 10),
		IdleTimeout:            getEnvAsInt("IDLE_TIMEOUT", 120),
		GCPProjectID:           getEnv("GCP_PROJECT_ID", ""),
		LogPath:                getEnv("LOG_PATH", "./log/isl-api-server.log"),
		LanguageDictionaryPath: getEnv("LANGUAGE_DICTIONARY_PATH", "./config/language_dictionary"),
		GeminiAPIKey:           getEnv("GEMINI_API_KEY", ""),
		AudioDBPath:            getEnv("AUDIO_DB_PATH", "./audiodb"),
	}

	return config, nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
