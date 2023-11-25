package repository

import (
	"database/sql"
	"errors"

	"diplom.com/cmd/gophermart/api/logger"
	"diplom.com/cmd/gophermart/models"
	_ "github.com/lib/pq"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db}
}

func (ur *UserRepository) Create(user models.User) error {

	log := logger.GetLogger()

	_, err := ur.db.Exec("INSERT INTO users (password, email) VALUES ($1, $2)", user.Password, user.Email)
	if err != nil {
		log.Error("Ошибка при выполнении запроса INSERT")
	}

	return err
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
        amount DECIMAL(10, 2) DEFAULT 0, -- Сумма заказа (по умолчанию 0)
        deduction DECIMAL(10, 2) DEFAULT 0, -- Сумма списания (по умолчанию 0)
        deduction_time TIMESTAMPTZ, -- Время списания
        CONSTRAINT valid_status CHECK (status IN ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED')) -- Проверка статуса заказа
    );
    `

	_, err := ur.db.Exec(query)
	return err
}

// В UserRepository добавь этот метод
func (ur *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	user := &models.User{}

	println("GetUserByEmail email", email)
	row := ur.db.QueryRow("SELECT email, password FROM users WHERE email = $1", email)

	err := row.Scan(&user.Email, &user.Password)
	println("после row GetUserByEmail", user.Email, user.Password)
	if err != nil {
		// Если нет соответствующих строк, вернуть пустое значение и ошибку
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		// Вернуть любую другую ошибку
		return nil, err
	}

	// Вернуть nil, если user.Email равен nil (возможно, стоит добавить другие проверки на корректность данных)
	if user.Email == "" {
		return nil, errors.New("user data is not valid")
	}

	return user, nil
}
