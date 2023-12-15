package config

import (
	"flag"
	"os"
)

type LConfig struct {
	RunAddress           string
	DatabaseURI          string
	AccrualSystemAddress string
	JWTSecretKey         string
}

var cfg *LConfig

func InitConfig() *LConfig {
	var (
		runAddress           string
		databaseURI          string
		accrualSystemAddress string
		jwtSecretKey         string
	)

	flag.StringVar(&runAddress, "a", "localhost:8081", "Адрес и порт запуска сервиса")
	flag.StringVar(&databaseURI, "d", "postgres://lew:qwert@localhost:5432/diplom?sslmode=disable", "Адрес подключения к базе данных")
	flag.StringVar(&accrualSystemAddress, "r", "localhost:1488", "Адрес системы расчёта начислений")
	flag.StringVar(&jwtSecretKey, "jwt", "Gazmaster358", "Секретный ключ JWT")

	runAddressEnv := os.Getenv("RUN_ADDRESS")
	if runAddressEnv != "" {
		runAddress = runAddressEnv
	}

	databaseURIEnv := os.Getenv("DATABASE_URI")
	if databaseURIEnv != "" {
		databaseURI = databaseURIEnv
	}

	accrualSystemAddressEnv := os.Getenv("ACCRUAL_SYSTEM_ADDRESS")
	if accrualSystemAddressEnv != "" {
		accrualSystemAddress = accrualSystemAddressEnv
	}

	jwtSecretKeyEnv := os.Getenv("JWT_SECRET_KEY")
	if jwtSecretKeyEnv != "" {
		jwtSecretKey = jwtSecretKeyEnv
	}

	flag.Parse()

	cfg = &LConfig{
		RunAddress:           runAddress,
		DatabaseURI:          databaseURI,
		AccrualSystemAddress: accrualSystemAddress,
		JWTSecretKey:         jwtSecretKey,
	}
	return cfg
}
