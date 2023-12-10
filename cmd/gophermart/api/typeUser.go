package api

import (
	"database/sql"
	"time"

	"diplom.com/cmd/gophermart/api/repository"
	"diplom.com/cmd/gophermart/api/repository/service"
	"diplom.com/cmd/gophermart/config"
	_ "github.com/lib/pq"

	"go.uber.org/zap"
)

type App struct {
	Logger *zap.Logger
	Config *config.LConfig
	DB     *sql.DB

	UserRepository      *repository.UserRepository
	UserService         *service.UserService
	Handler             UserHandler
	AuthCookieNameField string
	AuthCookieDuration  time.Duration
}

func (a *App) AuthCookieName() string {
	return "Gazmaster358"
}

func Init(logger *zap.Logger, cfg *config.LConfig, db *sql.DB) *App {
	return &App{
		Logger:         logger,
		Config:         cfg,
		DB:             db,
		UserRepository: repository.NewUserRepository(db),
	}
}
