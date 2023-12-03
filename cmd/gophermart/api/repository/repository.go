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
	_, err = ur.db.Exec("INSERT INTO users (password, user_email) VALUES ($1, $2)", user.Password, user.Email)
	if err != nil {
		return myerr.ErrInsertFailed
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
        id SERIAL PRIMARY KEY, -- Идентификатор токена
        user_email TEXT REFERENCES users(user_email), -- Ссылка на пользователя
        token TEXT NOT NULL, -- Токен, обязательное поле
        created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP -- Время создания токена
    );
    `

	_, err := ur.db.Exec(query)
	return err
}

func (ur *UserRepository) InsertToken(user_email string, token string) error {
	query := `
        INSERT INTO tokens (user_email, token)
        VALUES ($1, $2)
        RETURNING id;
    `

	var id int
	err := ur.db.QueryRow(query, user_email, token).Scan(&id)
	if err != nil {
		return err
	}

	return nil
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
		// Некорректные данные пользователя
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
