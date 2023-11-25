package api

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"time"

	"diplom.com/cmd/gophermart/api/logger"
	"diplom.com/cmd/gophermart/api/repository/service"
	"diplom.com/cmd/gophermart/models"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

func (mc *App) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	var user models.User
	log := logger.GetLogger()

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&user); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Error("RegisterUserHandler Ошибка при разборе тела запроса", zap.Error(err))
		return
	}

	// Проверить, не существует ли уже пользователь с таким же  email
	existingUser, _ := mc.UserRepository.GetUserByEmail(user.Email)
	if existingUser != nil {
		// Пользователь с таким email уже существует
		w.WriteHeader(http.StatusConflict)
		log.Error("Пользователь с таким email уже зарегистрирован")
		return
	}

	// Вызов метода сервиса для регистрации пользовател
	userService := service.NewUserService(mc.UserRepository)

	if err := userService.RegisterUser(user); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error("Ошибка аутентификации: неверное имя пользователя или пароль", zap.Error(err))
		return
	}

	// Создание нового запроса на аутентификацию с обновленными учетными данными
	authRequest, err := http.NewRequest("POST", "/api/user/login", nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error("Ошибка создания запроса на аутентификацию", zap.Error(err))
		return
	}

	// Передача обновленных учетных данных в теле запроса
	authRequestBody := map[string]string{
		"email":    user.Email,
		"password": user.Password,
	}

	authRequestBodyBytes, err := json.Marshal(authRequestBody)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error("Ошибка маршалинга учетных данных для запроса на аутентификацию", zap.Error(err))
		return
	}

	authRequest.Body = ioutil.NopCloser(bytes.NewReader(authRequestBodyBytes))
	authRequest.Header.Set("Content-Type", "application/json")

	// Выполнение запроса на аутентификацию
	authResponseRecorder := httptest.NewRecorder()
	mc.AuthenticateUserHandler(authResponseRecorder, authRequest)

	w.WriteHeader(authResponseRecorder.Code)
	w.Header().Set("Content-Type", authResponseRecorder.Header().Get("Content-Type"))
	w.Write(authResponseRecorder.Body.Bytes())

	// Вернуть успешный статус HTTP

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (mc *App) AuthenticateUserHandler(w http.ResponseWriter, r *http.Request) {
	log := logger.GetLogger()

	// Проверка заголовка Content-Type
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		log.Error("Неверный Content-Type. Ожидался application/json")
		return
	}
	log.Info("Запрос на аутентификацию", zap.Any("Headers", r.Header))

	// Разбор учетных данных пользователя из тела запроса
	var credentials models.User
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&credentials); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Error("AuthenticateUserHandler Ошибка при разборе тела запроса", zap.Error(err))
		return
	}

	// Вызов метода сервиса для аутентификации пользователя
	authToken, err := mc.UserService.AuthenticateUser(mc.UserRepository, credentials.Email, credentials.Password)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		log.Error("Ошибка аутентификации: неверная пара логин/пароль", zap.Error(err))
		return
	}

	// Установка токена аутентификации в куки
	cookie := http.Cookie{
		Name:     mc.AuthCookieNameField, // Используйте глобальную переменную для имени куки
		Value:    authToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(mc.AuthCookieDuration), // Используйте глобальную переменную для срока действия токена
		Secure:   true,                                  // Рекомендуется использовать только с HTTPS
	}

	w.Header().Set("Set-Cookie", cookie.String())
	log.Info("Аутентификация успешна")

	// Вернуть успешный статус HTTP
	w.WriteHeader(http.StatusOK)
}

func (mc *App) LoadOrders(w http.ResponseWriter, r *http.Request) {
	body := make([]byte, 1<<20) // Ограничение размера тела запроса до 1 МБ
	n, err := io.ReadFull(r.Body, body)
	if err != nil && err != io.ErrUnexpectedEOF {
		http.Error(w, "Ошибка чтения тела запроса", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	orderNumber := string(body[:n]) // Преобразование тела запроса в строку с номером заказа

	isValid := luhnCheck(orderNumber)
	if isValid {
		// Если номер валиден, сохраняем в базу данных
		err := mc.SaveOrderNumber(mc.DB, orderNumber)
		if err != nil {
			http.Error(w, "Ошибка сохранения номера заказа", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Номер заказа успешно сохранен в базе данных"))
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Номер заказа не прошел проверку по методу Луна"))
	}
}

// Функция для проверки номера по алгоритму Луна
func luhnCheck(number string) bool {
	sum := 0
	shouldDouble := false

	// Обратный цикл по цифрам номера
	for i := len(number) - 1; i >= 0; i-- {
		digit := int(number[i] - '0')

		// Удвоение каждой второй цифры
		if shouldDouble {
			if digit *= 2; digit > 9 {
				digit -= 9
			}
		}

		// Суммирование цифр
		sum += digit
		shouldDouble = !shouldDouble
	}

	// Номер валиден, если сумма кратна 10
	return sum%10 == 0
}

func (mc *App) Route() *chi.Mux {

	r := chi.NewRouter()

	r.With(mc.Authenticate).Post("/api/user/orders", mc.LoadOrders) // Подключение middleware для проверки аутентификации к маршруту /api/user/orders

	r.Post("/api/user/register", mc.RegisterUserHandler)
	r.Post("/api/user/login", mc.AuthenticateUserHandler)

	return r
}

// Middleware для проверки аутентификации пользователя
func (mc *App) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем данные из тела запроса
		var user models.User
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&user); err != nil {
			http.Error(w, "Ошибка при чтении данных пользователя", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Предполагая, что login и password приходят в структуре User
		authenticated, err := mc.UserService.AuthenticateUser(mc.UserRepository, user.Email, user.Password)
		if err != nil {
			http.Error(w, "Ошибка при проверке аутентификации", http.StatusInternalServerError)
			return
		}

		if authenticated != "" {
			http.Error(w, "Неавторизованный доступ", http.StatusUnauthorized)
			return
		}

		// Если аутентификация прошла успешно, продолжаем выполнение следующего обработчика
		next.ServeHTTP(w, r)
	})
}
