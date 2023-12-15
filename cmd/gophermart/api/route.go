package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

func (mc *App) Route() *chi.Mux {
	r := chi.NewRouter()

	// Создание логгера zap
	logger, err := zap.NewProduction() // Используйте NewDevelopment() вместо NewProduction() для разработки
	if err != nil {
		standardLogger := zap.NewExample() // Создание стандартного логгера для вывода ошибки
		standardLogger.Error("Ошибка создания логгера", zap.Error(err))
	}
	defer logger.Sync() // Синхронизация логгера перед завершением работы

	// Middleware для аутентификации пользователя
	authMiddleware := mc.TokenAuth // TokenAuth - middleware

	// Маршруты, требующие аутентификации через токен....
	r.Route("/api/user", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)

			// POST запрос для создания заказа
			r.Post("/orders", mc.LoadOrders)

			// GET запрос для получения заказов
			r.Get("/orders", mc.GetUserOrdersHandler)
		})
	})

	// Маршруты без аутентификац
	r.Post("/api/user/register", mc.RegisterUserHandler)
	r.Post("/api/user/login", mc.AuthenticateUserHandler)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Неправильный путь запроса: %s\n", r.URL.Path)
		logger.Error("Неправильный путь запроса", zap.String("path", r.URL.Path))

	})

	return r
}
