package repository

import (
	"database/sql"
	"errors"
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
	return &UserRepository{db}
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
	_, err = ur.db.Exec("INSERT INTO users (password, email) VALUES ($1, $2)", user.Password, user.Email)
	if err != nil {
		return myerr.ErrInsertFailed
	}

	return nil
}

func (ur *UserRepository) CreateTableUsers() error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY, -- Идентификатор пользователя
		email TEXT NOT NULL UNIQUE, -- Электронная почта, обязательное поле, уникальное значение
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
        user_id INT REFERENCES users(id), -- Ссылка на пользователя
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

// В UserRepository добавь этот метод

func (ur *UserRepository) GetUserByEmail(email string, password string) (*models.User, error) {
	user := &models.User{
		Email:    email,
		Password: password,
	}

	row := ur.db.QueryRow("SELECT email, password FROM users WHERE email = $1", email)

	err := row.Scan(&user.Email, &user.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			// Пользователь не найден - записываем сообщение в лог
			log.Println("Пользователь не найден для email:", email)
			return nil, errors.New("user not found")
		}
		// Вернуть любую другую ошибку
		return nil, err
	}

	if user.Email == "" {
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

	_, err := ur.db.Exec("UPDATE users SET password = $1 WHERE email = $2", newPassword, email)
	if err != nil {
		log.Error("Ошибка при выполнении запроса UPDATE")
	}

	return err
}
