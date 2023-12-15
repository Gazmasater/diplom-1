package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

func InitLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	zapLogger, err := config.Build()
	if err != nil {
		// Логгировать ошибку
		logError := zap.NewExample().Sugar()
		logError.Errorf("Ошибка инициализации логгера: %v", err)
		return nil, err
	}
	log = zapLogger
	return zapLogger, nil
}

// Получить логгер
func GetLogger() *zap.Logger {
	return log
}
