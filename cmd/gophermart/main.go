package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"diplom.com/cmd/gophermart/api"
	"diplom.com/cmd/gophermart/api/logger"
	"diplom.com/cmd/gophermart/config"
	"go.uber.org/zap"
)

func main() {
	// Инициализация логгера
	logger, err := logger.InitLogger()
	if err != nil {
		// обработка ошибки инициализации логгера
		panic(err)
	}

	// Устанавливаем соединение с базой данных PostgreSQL
	db, err := sql.Open("postgres", "user=lew dbname=diplom password=qwert host=localhost sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Проверяем соединение с базой данных
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected to the database")

	// Инициализируем конфиг и другие необходимые параметры, если нужно
	cfg := &config.L_Config{} // ваша конфигурация

	// Инициализируем приложение с переданным соединением с базой данных
	app := api.Init(logger, cfg, db)

	if err := app.UserRepository.CreateTableUsers(); err != nil {
		logger.Error("Ошибка создания таблицы пользователей", zap.Error(err))
		// Обработка ошибки создания таблицы пользователей
	}

	if err := app.UserRepository.CreateTableOrders(); err != nil {
		logger.Error("Ошибка создания таблицы пользователей", zap.Error(err))
		// Обработка ошибки создания таблицы пользователей
	}

	router := app.Route()

	go func() {
		if err := http.ListenAndServe(":8080", router); err != nil {
			// Если сервер не запущен
			logger.Error("Ошибка запуска сервера", zap.Error(err))
		}
	}()

	// Логирование успешного запуска сервера
	logger.Info("Сервер успешно запущен на порту 8080")

	// Для того чтобы приложение не завершалось сразу
	select {}
}
