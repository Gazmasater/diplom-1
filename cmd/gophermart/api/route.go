package api

import "github.com/go-chi/chi"

func (mc *App) Route() *chi.Mux {
	r := chi.NewRouter()

	// Middleware для аутентификации пользователя
	authMiddleware := mc.TokenAuth // TokenAuth - middleware
	r.Use(BadRequestMiddleware)

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
