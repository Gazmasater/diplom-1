package api

import (
	"time"

	"diplom.com/cmd/gophermart/api/logger"
	"diplom.com/cmd/gophermart/api/myerr"
	"go.uber.org/zap"
)

func (app *App) SaveOrderNumber(orderNumber string) error {
	log := logger.GetLogger()

	// Проверка на уникальность номера заказа перед вставкой
	var count int
	err := app.DB.QueryRow("SELECT COUNT(*) FROM orders WHERE order_number = $1", orderNumber).Scan(&count)
	if err != nil {
		log.Error("Ошибка при проверке уникальности номера заказа", zap.Error(err))
		return err
	}
	if count > 0 {
		return myerr.ErrOrderNumberNotUnique // Возвращаем созданную ошибку
	}

	// Запрос на вставку номера заказа в базу данных
	query := "INSERT INTO orders (order_number, status, created_at) VALUES ($1, 'NEW', $2)"
	createdAt := time.Now().Format(time.RFC3339) // Форматируем текущее время в формат RFC3339
	_, err = app.DB.Exec(query, orderNumber, createdAt)
	if err != nil {
		log.Error("Ошибка при сохранении номера заказа в базе данных", zap.Error(err))
		return err
	}

	return nil
}
