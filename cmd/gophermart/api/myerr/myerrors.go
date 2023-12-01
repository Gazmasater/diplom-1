package myerr

import "errors"

// Custom errors
var (
	ErrUserAlreadyExists    = errors.New("пользователь уже зарегистрирован")
	ErrInsertFailed         = errors.New("ошибка при выполнении запроса INSERT")
	ErrOrderNumberNotUnique = errors.New("ошибка при проверке уникальности номера заказа")
)
