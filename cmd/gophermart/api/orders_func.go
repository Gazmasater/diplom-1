package api

import (
	"database/sql"

	"diplom.com/cmd/gophermart/api/logger"
	"go.uber.org/zap"
)

func (app *App) SaveOrderNumber(db *sql.DB, orderNumber string) error {
	log := logger.GetLogger()

	app = &App{

		DB: db, // ваш экземпляр *sql.DB

	}

	// Здесь выполняется SQL-запрос для сохранения номера заказа в базу данных
	query := "INSERT INTO orders (order_number, status) VALUES ($1, 'NEW')"
	_, err := db.Exec(query, orderNumber)
	if err != nil {
		log.Error("Ошибка при сохранении номера заказа в базе данных", zap.Error(err))
		return err
	}

	return nil
}
