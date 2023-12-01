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

	router := app.Route()

	// go func() {
	// 	for {
	// 		// Проверяем базу данных на наличие заказов со статусом NEW
	// 		var orderNumber string
	// 		err := db.QueryRow("SELECT order_number FROM orders WHERE status = 'NEW' LIMIT 1").Scan(&orderNumber)
	// 		if err != nil {
	// 			log.Println("Ошибка при проверке статуса заказа:", err)
	// 			// Обработка ошибки проверки статуса заказа
	// 			time.Sleep(5 * time.Second) // Пауза перед следующей попыткой
	// 			continue
	// 		}

	// 		// Если есть заказ со статусом NEW, выполняем POST-запрос
	// 		if orderNumber != "" {
	// 			// Создаем JSON-данные для запроса
	// 			jsonData := `{"order": "` + orderNumber + `"}`

	// 			// Выполняем POST-запрос
	// 			resp, err := http.Post("http://localhost:8081/api/orders", "application/json", bytes.NewBuffer([]byte(jsonData)))
	// 			if err != nil {
	// 				log.Println("Ошибка при выполнении POST-запроса:", err)
	// 				// Обработка ошибки POST-запроса
	// 			} else {

	// 				// Проверяем статус ответа
	// 				if resp.StatusCode != http.StatusOK {
	// 					log.Println("Ошибка: получен некорректный статус ответа:", resp.StatusCode)
	// 					// Обработка ошибки некорректного статуса ответа
	// 				} else {
	// 					log.Println("Выполнен POST-запрос для заказа:", orderNumber)
	// 				}
	// 			}

	// 			// Помечаем заказ как обработанный, например, обновляем статус
	// 			_, err = db.Exec("UPDATE orders SET status = 'PROCESSED' WHERE order_number = $1", orderNumber)
	// 			if err != nil {
	// 				log.Println("Ошибка при обновлении статуса заказа:", err)
	// 				// Обработка ошибки обновления статуса заказа
	// 			}
	// 		}

	// 		// Пауза перед следующей проверкой
	// 		time.Sleep(5 * time.Second)
	// 	}
	// }()

	go func() {

		if err := http.ListenAndServe(":8080", router); err != nil {
			// Если сервер не запущен
			logger.Error("Ошибка запуска сервера", zap.Error(err))
		}

	}()

	go func() {
		defer wg.Done()
		orderNumber := fmt.Sprintf("%d", 12345678903) // преобразование числа в строку

		// Создание JSON-структуры
		jsonData := map[string]string{"order": orderNumber}
		jsonValue, _ := json.Marshal(jsonData)

		req, err := http.NewRequest("POST", "http://localhost:8080/api/user/orders", bytes.NewBuffer(jsonValue))
		if err != nil {
			fmt.Println("Ошибка при создании запроса:", err)
			return
		}

		req.Header.Set("Authorization", "Bearer Gazmaster358")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Ошибка при выполнении запроса:", err)
			return
		}
		defer resp.Body.Close()

		fmt.Println("Ответ:", resp.Status)
	}()

	// Логирование успешного запуска сервера
	logger.Info("Сервер успешно запущен на порту 8080")

	time.Sleep(10 * time.Second)

	// Для того чтобы приложение не завершалось сразу
	select {}
}
