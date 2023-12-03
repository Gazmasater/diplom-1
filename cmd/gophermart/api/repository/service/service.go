package service

import (
	"errors"
	"os"

	"time"

	"diplom.com/cmd/gophermart/api/logger"
	"diplom.com/cmd/gophermart/api/myerr"
	"diplom.com/cmd/gophermart/api/repository"
	"diplom.com/cmd/gophermart/models"
	"github.com/dgrijalva/jwt-go"
	"go.uber.org/zap"
)

type UserService struct {
	userRepository *repository.UserRepository
}

// Конструктор для UserService, принимающий userRepository в качестве аргумента
func NewUserService(userRepository *repository.UserRepository) *UserService {
	return &UserService{
		userRepository: userRepository,
		// инициализация других полей, если есть
	}
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

	if err := us.userRepository.Create(user); err != nil {
		if errors.Is(err, myerr.ErrUserAlreadyExists) {
			// Обработка ошибки "Пользователь уже зарегистрирован"
			log.Error("RegisterUser Пользователь с таким email уже зарегистрирован\n", zap.Error(err))

		} else if errors.Is(err, myerr.ErrInsertFailed) {
			// Обработка ошибки "Ошибка при выполнении запроса INSERT"
			log.Error("Ошибка при выполнении запроса INSERT для пользователя", zap.Error(err))

		} else {
			// Обработка других ошибок
			log.Error("Необработанная ошибка при создании пользователя", zap.Error(err))

		}
		return err
	}

	return nil
}

func (us *UserService) AuthenticateUser(userRepository *repository.UserRepository, login string, password string) (string, error) {
	log := logger.GetLogger()

	// Получение пользователя по логину
	user, err := userRepository.GetUserByEmail(login, password)
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
	secretKey := []byte(os.Getenv("Gazmaster358")) // Используйте переменную окружения для хранения секретного ключа
	signedToken, err := token.SignedString(secretKey)
	if err != nil {
		log.Error("Ошибка при подписи токена аутентификации", zap.Error(err))
		return "", err
	}

	// Сохранение токена в базе данных
	err = userRepository.InsertToken(user.Email, signedToken)
	if err != nil {
		log.Error("Ошибка при сохранении токена в базе данных", zap.Error(err))
		return "", err
	}

	return signedToken, nil
}
