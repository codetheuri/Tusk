package database

import (
	"context"
	"fmt"
	"time"

	"github.com/codetheuri/todolist/config"
	appErrors "github.com/codetheuri/todolist/pkg/errors"
	"github.com/codetheuri/todolist/pkg/logger"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func NewGoRMDB(cfg *config.Config, log logger.Logger) (*gorm.DB, error) {

	newLogger := NewGormLogger(log)
	db, err := config.ConnectDB()
	if db != nil {

		db.Logger = newLogger.LogMode(gormlogger.Info)
	}

	if err != nil {
		log.Error("failed to connect to database", err, "dsn_info", fmt.Sprintf("user: %s, host: %s, port: %s, dbname: %s", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName))
		return nil, appErrors.DatabaseError("failed tp connect to database", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Error("failed to get undelying sql.DB", err)
	}
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.DBConnMaxLifetime) * time.Second)

	if err = sqlDB.Ping(); err != nil {
		log.Error("database is unreachable", err)
		return nil, appErrors.DatabaseError("database is unreachable", err)
	}
	log.Info("Database connected successfully ")
	return db, nil
}

type GormLogger struct {
	logger logger.Logger
}

func NewGormLogger(log logger.Logger) GormLogger {
	return GormLogger{
		logger: log,
	}
}
func (gl GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return gl
}

func (gl GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	gl.logger.Info(msg, data...)
}
func (gl GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	gl.logger.Warn(msg, data...)
}

func (gl GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	var actualErr error
	var cleanedData []interface{}

	for _, item := range data {
		if e, ok := item.(error); ok {
			actualErr = e
		} else {
			cleanedData = append(cleanedData, item)
		}
	}

	gl.logger.Error(msg, actualErr, cleanedData...)
}
func (gl GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	sql, rowsAffected := fc()
	duration := time.Since(begin)
	fields := []interface{}{
		"duration", duration,
		"rows_affected", rowsAffected,
		"sql", sql,
	}
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			gl.logger.Debug("GORM Trace", fields...)
		} else {
			gl.logger.Error("GORM Trace", err, fields...)
		}
	} else {
		gl.logger.Debug("GORM Trace", fields...)
	}
}
