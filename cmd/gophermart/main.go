package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"diplom.com/cmd/gophermart/api"
	"diplom.com/cmd/gophermart/api/logger"
	"diplom.com/cmd/gophermart/config"
	"go.uber.org/zap"
)

func main() {

	var wg sync.WaitGroup
	wg.Add(1) // Добавление в WaitGroup

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

	if err := app.UserRepository.CreateTableTokens(); err != nil {
		logger.Error("Ошибка создания таблицы пользователей", zap.Error(err))
		// Обработка ошибки создания таблицы пользователей
	}

	router := app.Route()

	go func() {
		defer wg.Done()

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			err := updateOrdersTable(db)
			if err != nil {
				logger.Error("Ошибка при обновлении таблицы orders", zap.Error(err))
			}
		}
	}()

	go func() {
		if err := http.ListenAndServe(":8080", router); err != nil {
			logger.Error("Ошибка запуска сервера", zap.Error(err))
		}
	}()

	logger.Info("Сервер успешно запущен на порту 8080")
	select {}

}

func sendPostRequest(orderNumber string) (string, error) {
	url := "http://localhost:8081/api/orders"
	requestBody := []byte(fmt.Sprintf(`{"order": "%s"}`, orderNumber))

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("неверный статус от сервера: %d", resp.StatusCode)
	}

	return orderNumber, nil
}

func updateOrdersTable(db *sql.DB) error {
	// Получение номера заказа из базы данных
	var orderNumber string
	err := db.QueryRow("SELECT order_number FROM orders WHERE status = 'NEW'").Scan(&orderNumber)
	if err != nil {
		if err == sql.ErrNoRows {
			// Если нет статуса "NEW", возвращаем nil
			return nil
		}
		return err
	}

	println("updateOrdersTable orderNumber ", orderNumber)
	orderNumber, err = sendPostRequest(orderNumber)
	if err != nil {
		return err
	}

	// Формирование GET запроса для получения статуса
	url := fmt.Sprintf("http://localhost:8081/api/orders/%s", orderNumber)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("неверный статус от сервера: %d", resp.StatusCode)
	}

	// Декодирование ответа и получение статуса и accrual
	var responseBody struct {
		Status  string `json:"status"`
		Accrual int    `json:"accrual"` // или другой тип данных accrual
		// Добавьте другие поля, если они есть в ответе
	}

	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	if err != nil {
		return err
	}

	// Обновление базы данных на основе полученных данных
	_, err = db.Exec("UPDATE orders SET status = $1, accrual = $2 WHERE order_number = $3", responseBody.Status, responseBody.Accrual, orderNumber)
	if err != nil {
		return err
	}

	return nil
}
