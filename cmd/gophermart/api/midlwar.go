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
