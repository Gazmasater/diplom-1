package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	"diplom.com/cmd/gophermart/api/myerr"

	"diplom.com/cmd/gophermart/api/logger"
	"diplom.com/cmd/gophermart/models"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (ur *UserRepository) Create(user models.User) error {
	log := logger.GetLogger()

	// Проверяем, существует ли пользователь с указанным email
	existingUser, err := ur.GetUserByEmail(user.Email, user.Password)
	if err != nil {
		log.Error("Такого пользователя нет, идем дальше", zap.Error(err))

	}

	// Если пользователь уже существует, возвращаем ошибку о том, что пользователь уже зарегистрирован
	if existingUser != nil {
		return myerr.ErrUserAlreadyExists
	}

	// Обновляем пароль пользователя
	err = ur.UpdatePassword(user.Email, user.Password)
	if err != nil {
		log.Error("Ошибка при обновлении пароля", zap.Error(err))
		return err
	}

	// Добавляем нового пользователя в базу данных
	_, err = ur.db.Exec("INSERT INTO public.users (password, user_email) VALUES ($1, $2)", user.Password, user.Email)
	if err != nil {
		return err
	}

	return nil
}

func (ur *UserRepository) CreateTableUsers() error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY, -- Идентификатор пользователя
		user_email TEXT NOT NULL UNIQUE, -- Электронная почта, обязательное поле, уникальное значение
		password TEXT NOT NULL, -- Пароль, обязательное поле
		balance DECIMAL(10, 2) DEFAULT 0 -- Баланс пользователя (по умолчанию 0)
	);
	`

	_, err := ur.db.Exec(query)
	return err
}

func (ur *UserRepository) CreateTableOrders() error {
	query := `
    CREATE TABLE IF NOT EXISTS orders (
		id SERIAL PRIMARY KEY, -- Идентификатор заказа
		user_email TEXT REFERENCES users(user_email), -- Ссылка на пользователя по электронной почте
		order_number TEXT NOT NULL, -- Номер заказа, обязательное поле
		status TEXT NOT NULL, -- Статус заказа, обязательное поле
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP, -- Время создания заказа
		accrual DECIMAL(10, 2) DEFAULT 0, -- Сумма начисления (по умолчанию 0)
		deduction DECIMAL(10, 2) DEFAULT 0, -- Сумма списания (по умолчанию 0)
		deduction_time TIMESTAMPTZ, -- Время списания
		CONSTRAINT valid_status CHECK (status IN ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED')) -- Проверка статуса заказа
	);
	
    `

	_, err := ur.db.Exec(query)
	return err
}

func (ur *UserRepository) CreateTableTokens() error {
	query := `
        CREATE TABLE IF NOT EXISTS tokens (
            id SERIAL PRIMARY KEY,
            user_email TEXT UNIQUE, -- Указываем уникальность для user_email
            token TEXT NOT NULL,
            created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
            expiration_time TIMESTAMPTZ
        );
    `

	_, err := ur.db.Exec(query)
	return err
}

// В UserRepository добавь этот метод

func (ur *UserRepository) GetUserByEmail(email string, password string) (*models.User, error) {
	fmt.Println("Trying to enter GetUserByEmail function") // Добавьте эту строку

	user := &models.User{
		Email:    email,
		Password: password,
	}

	row := ur.db.QueryRow("SELECT user_email, password FROM users WHERE user_email = $1", email)

	err := row.Scan(&user.Email, &user.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			// Пользователь не найден
			log.Println("Пользователь не найден для email:", email)
			return nil, errors.New("user not found")
		}
		// Ошибка при выполнении запроса к базе данных
		log.Println("Ошибка при выполнении запроса к базе данных:", err)
		return nil, err
	}

	if user.Email == "" {

		log.Println("Некорректные данные пользователя")
		return nil, errors.New("user data is not valid")
	}

	return user, nil
}

func (ur *UserRepository) UpdatePassword(email, newPassword string) error {
	log := logger.GetLogger()

	if newPassword == "" {
		// Новый пароль не предоставлен, ничего не меняем
		return nil
	}

	_, err := ur.db.Exec("UPDATE users SET password = $1 WHERE user_email = $2", newPassword, email)
	if err != nil {
		log.Error("Ошибка при выполнении запроса UPDATE")
	}

	return err
}

func (ur *UserRepository) InsertToken(userEmail string, token string, expirationTime string) error {
	// Удаление всех строк с указанным user_email
	deleteQuery := `
        DELETE FROM tokens
        WHERE user_email = $1
    `
	_, err := ur.db.Exec(deleteQuery, userEmail)
	if err != nil {
		return err
	}

	insertQuery := `
        INSERT INTO tokens (user_email, token, expiration_time)
        VALUES ($1, $2, $3)
    `
	_, err = ur.db.Exec(insertQuery, userEmail, token, expirationTime)
	return err
}

func (ur *UserRepository) GetOrdersWithUserEmail(userEmail string) ([]struct {
	OrderNumber string `json:"order_number"`
	UserEmail   string `json:"user_email"`
}, error) {
	query := `
        SELECT order_number, user_email
        FROM orders
        WHERE user_email = $1
    `

	rows, err := ur.db.Query(query, userEmail)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []struct {
		OrderNumber string `json:"order_number"`
		UserEmail   string `json:"user_email"`
	}
	for rows.Next() {
		var order struct {
			OrderNumber string
			UserEmail   string
		}
		if err := rows.Scan(&order.OrderNumber, &order.UserEmail); err != nil {
			return nil, err
		}
		orders = append(orders, struct {
			OrderNumber string `json:"order_number"`
			UserEmail   string `json:"user_email"`
		}{
			OrderNumber: order.OrderNumber,
			UserEmail:   order.UserEmail,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}
