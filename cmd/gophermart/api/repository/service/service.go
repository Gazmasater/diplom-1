package service

import (
	"errors"
	"os"

	"time"

	"diplom.com/cmd/gophermart/api/logger"
	"diplom.com/cmd/gophermart/api/repository"
	"diplom.com/cmd/gophermart/models"
	"github.com/dgrijalva/jwt-go"
	"go.uber.org/zap"
)

type UserService struct {
	userRepository *repository.UserRepository
}

func NewUserService(userRepository *repository.UserRepository) *UserService {
	println("NewUserService")
	return &UserService{userRepository}
}

func (us *UserService) RegisterUser(user models.User) error {
	log := logger.GetLogger()

	// Создание таблицы пользователей
	if err := us.userRepository.CreateTableUsers(); err != nil {
		log.Error("Ошибка при создании таблицы пользователей", zap.Error(err))
		return err
	}

	// Создание таблицы заказов
	if err := us.userRepository.CreateTableOrders(); err != nil {
		log.Error("Ошибка при создании таблицы заказов", zap.Error(err))
		return err
	}

	// Создание пользователя
	if err := us.userRepository.Create(user); err != nil {
		log.Error("Ошибка при выполнении запроса INSERT для пользователя", zap.Error(err))
		return err
	}

	return nil
}

func (us *UserService) AuthenticateUser(userRepository *repository.UserRepository, login string, password string) (string, error) {
	log := logger.GetLogger()

	// Получение пользователя по логину
	user, err := us.userRepository.GetUserByEmail(login)
	if err != nil {
		log.Error("Ошибка при получении пользователя", zap.Error(err))
		return "", err
	}

	// Проверка пароля
	if user == nil || user.Password != password {
		return "", errors.New("неверная пара логин/пароль")
	}

	// Создание токена аутентификации (JWT)
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["user_id"] = user.Email
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix() // Пример: токен действителен 24 часа

	// Подпись токена с использованием секретного ключа из переменной окружения
	secretKey := []byte(os.Getenv("JWT_SECRET_KEY")) // Используйте переменную окружения для хранения секретного ключа
	signedToken, err := token.SignedString(secretKey)
	if err != nil {
		log.Error("Ошибка при подписи токена аутентификации", zap.Error(err))
		return "", err
	}

	return signedToken, nil
}
