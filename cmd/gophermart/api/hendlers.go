package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"net/http"
	"net/http/httptest"
	"time"

	"diplom.com/cmd/gophermart/api/logger"
	"diplom.com/cmd/gophermart/api/myerr"
	"diplom.com/cmd/gophermart/api/repository/service"
	"diplom.com/cmd/gophermart/models"
	"github.com/dgrijalva/jwt-go"
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
	defer r.Body.Close()
	println("user", user.Email, user.Password)

	existingUser, _ := mc.UserRepository.GetUserByEmail(user.Email, user.Password)

	if existingUser != nil {
		w.WriteHeader(http.StatusConflict)
		log.Error("RegisterUserHandler Пользователь с таким email уже зарегистрирован\n")
		return
	}

	userService := service.NewUserService(mc.UserRepository)
	if err := userService.RegisterUser(user); err != nil {
		if errors.Is(err, myerr.ErrUserAlreadyExists) {
			w.WriteHeader(http.StatusConflict)
			fmt.Fprintf(w, "Пользователь с таким email уже зарегистрирован\n ")
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		log.Error("Ошибка регистрации пользователя", zap.Error(err))
		return
	}

	authRequest, err := http.NewRequest("POST", "/api/user/login", nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error("Ошибка создания запроса на аутентификацию", zap.Error(err))
		return
	}

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

	authRequest.Body = io.NopCloser(bytes.NewReader(authRequestBodyBytes))
	authRequest.Header.Set("Content-Type", "application/json")

	authResponseRecorder := httptest.NewRecorder()
	mc.AuthenticateUserHandler(authResponseRecorder, authRequest)

	w.WriteHeader(authResponseRecorder.Code)
	w.Header().Set("Content-Type", authResponseRecorder.Header().Get("Content-Type"))
	w.Write(authResponseRecorder.Body.Bytes())

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
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "Missing authorization header\n")
		return
	}

	bearerToken := strings.Split(authHeader, " ")
	if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "Invalid token format")
		return
	}

	tokenString := bearerToken[1]

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte("Gazmaster358"), nil // Замените на ваш секретный ключ
	})

	if err != nil || !token.Valid {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "Invalid token")
		return
	}

	claims, _ := token.Claims.(jwt.MapClaims)
	userID, _ := claims["user_id"].(string)
	println("LoadOrders userID", userID)

	body := make([]byte, 1<<20) // Ограничение размера тела запроса до 1 МБ
	n, err := io.ReadFull(r.Body, body)
	if err != nil && err != io.ErrUnexpectedEOF {
		http.Error(w, "Ошибка чтения тела запроса", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	orderNumber := extractOrderNumberFromString(string(body[:n])) // Извлечение номера заказа из тела запроса

	if orderNumber == "" {
		http.Error(w, "Номер заказа не найден в теле запроса", http.StatusBadRequest)
		return
	}

	isValid := luhnCheck(orderNumber)
	if isValid {
		// Если номер валиден, сохраняем в базу данных
		err := mc.SaveOrderNumber(orderNumber)
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

func extractOrderNumberFromString(input string) string {
	regexPattern := `\"order\":\s*\"(\d+)\"`
	re := regexp.MustCompile(regexPattern)
	matches := re.FindStringSubmatch(input)
	if len(matches) != 2 {
		return "" // Возвращаем пустую строку в случае отсутствия совпадения
	}

	return matches[1] // Возвращаем найденный номер заказа
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

	// Middleware для аутентификации пользователя
	authMiddleware := mc.TokenAuth // TokenAuth - middleware

	// Маршруты, требующие аутентификации через токен
	r.Route("/api/user", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)

			// POST запрос для создания заказа
			r.Post("/orders", mc.LoadOrders)

			// GET запрос для получения заказов
			r.Get("/orders", mc.GetUserOrdersHandler)
		})
	})

	// Маршруты без аутентификации
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

func (mc *App) GetUserOrdersHandler(w http.ResponseWriter, r *http.Request) {
	userEmail := r.URL.Query().Get("user_email")
	if userEmail == "" {
		http.Error(w, "Не указан адрес электронной почты", http.StatusBadRequest)
		return
	}

	query := `
    SELECT id, order_number, status, created_at, accrual, deduction, deduction_time
    FROM orders
    WHERE user_email = $1
    `

	rows, err := mc.DB.Query(query, userEmail)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		if err := rows.Scan(&order.ID, &order.OrderNumber, &order.Status, &order.CreatedAt, &order.Accrual, &order.Deduction, &order.DeductionTime); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		orders = append(orders, order)

		// Вывод номера заказа
		fmt.Println("Номер заказа:", order.OrderNumber)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if len(orders) == 0 {
		fmt.Println("У пользователя нет заказов")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
		return
	}

	if err := json.NewEncoder(w).Encode(orders); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (mc *App) TokenAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userEmail := r.URL.Query().Get("user_email")
		fmt.Println("Получен адрес электронной почты:", userEmail) // Отладочный принт

		if userEmail == "" {
			http.Error(w, "TokenAuth: Не указан адрес электронной почты", http.StatusBadRequest)
			return
		}

		// Получаем токен из базы данных для указанного email
		var token models.Token
		query := "SELECT id, user_email, token, created_at, expiration_time FROM tokens WHERE user_email = $1"

		err := mc.DB.QueryRow(query, userEmail).Scan(
			&token.ID, &token.UserEmail, &token.Token, &token.CreatedAt, &token.ExpirationTime,
		)
		if err != nil {

			http.Error(w, "TokenAuth: Токен не найден или истек", http.StatusUnauthorized)
			return
		}

		// Сравниваем время истечения с текущим временем
		if token.ExpirationTime.Before(time.Now()) {

			http.Error(w, "TokenAuth: Токен истёк", http.StatusUnauthorized)
			return
		}

		// Если токен валиден, передаем управление следующему обработчику
		next.ServeHTTP(w, r)
	})
}
