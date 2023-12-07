package api

import (
	"fmt"
	"net/http"
	"time"

	"diplom.com/cmd/gophermart/models"
)

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

func BadRequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Здесь проверяем, соответствует ли запрос определенным обработчикам
		// Если запрос не соответствует ни одному из обработчиков, устанавливаем статус 400
		status := http.StatusBadRequest

		// Проверяем метод запроса и путь
		switch r.Method {
		case "GET":
			// Обработчик для метода GET
			if r.URL.Path == "/specific-get-endpoint" {
				status = http.StatusOK
			}
		case "POST":
			// Обработчик для метода POST
			if r.URL.Path == "/specific-post-endpoint" {
				status = http.StatusOK
			}
		// Добавьте другие методы и пути по мере необходимости

		default:
			// Если запрос не соответствует ожидаемым обработчикам, устанавливаем статус 400
			status = http.StatusBadRequest
		}

		// Если статус 400, возвращаем его, иначе передаем обработку следующему обработчику
		if status == http.StatusBadRequest {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Неверный запрос\n")
			return
		}

		next.ServeHTTP(w, r)
	})
}
