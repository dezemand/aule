package config

import "github.com/dezemandje/aule/internal/database"

type DBConfig = database.Config

func NewDBConfigFromEnv() DBConfig {
	return DBConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "aule"),
		Password: getEnv("DB_PASSWORD", "aule"),
		DBName:   getEnv("DB_NAME", "aule"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
}
