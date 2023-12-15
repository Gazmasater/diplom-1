package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type UserHandler interface {
	RegisterUserHandler(w http.ResponseWriter, r *http.Request)
	AuthenticateUserHandler(w http.ResponseWriter, r *http.Request)
	Route() *chi.Mux
}
