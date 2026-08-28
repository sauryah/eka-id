package config

import (
	"os"
	"strconv"
)

type Config struct {
	Environment        string
	ServerPort         string
	ServerHost         string
	CorsAllowedOrigins string
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	DBSSLMode          string
	RedisHost          string
	RedisPort          string
	RedisPassword      string
	JWTSecret          string
	VerifyURLPrefix    string
	DataPath           string
}

func Load() *Config {
	return &Config{
		Environment:        getEnv("ENVIRONMENT", "development"),
		ServerPort:         getEnv("SERVER_PORT", "8080"),
		ServerHost:         getEnv("SERVER_HOST", "0.0.0.0"),
		CorsAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "*"),
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnv("DB_PORT", "5433"),
		DBUser:             getEnv("DB_USER", "eka_admin"),
		DBPassword:         getEnv("DB_PASSWORD", "eka_secure_dev_pass_2026"),
		DBName:             getEnv("DB_NAME", "eka_id"),
		DBSSLMode:          getEnv("DB_SSL_MODE", "disable"),
		RedisHost:          getEnv("REDIS_HOST", "localhost"),
		RedisPort:          getEnv("REDIS_PORT", "6380"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		JWTSecret:          getEnv("JWT_SECRET", "eka_jwt_dev_super_secret_signing_key_32bytes_minimum_length_2026"),
		VerifyURLPrefix:    getEnv("VERIFY_URL_PREFIX", "http://localhost:3000/verify"),
		DataPath:           getEnv("DATA_PATH", "./data/eka_database.json"),
	}
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}