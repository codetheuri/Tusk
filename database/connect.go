package database

// This file owns connecting to PostgreSQL. It sits beside the migration runner
// deliberately: both are "the database layer", and the previous
// internal/platform/ tier existed to hold exactly one file. Being here rather
// than under internal/ also means an application built on Tusk can open a
// connection configured the same way the framework does.

import (
	"context"
	"fmt"
	"time"

	"github.com/codetheuri/tusk/v2/config"
	"github.com/codetheuri/tusk/v2/pkg/logger"
	"github.com/codetheuri/tusk/v2/pkg/tenant"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func Connect(cfg *config.Config, log logger.Logger) (*gorm.DB, error) {
	newLogger := NewGormLogger(log)

	var db *gorm.DB
	var err error

	switch cfg.DBDriver {
	case "postgres", "pgsql":
		db, err = gorm.Open(postgres.Open(cfg.DbURL), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER: %s", cfg.DBDriver)
	}

	if db != nil {
		db.Logger = newLogger.LogMode(gormlogger.Info)
	}

	if err != nil {
		log.Error("failed to connect to database", err, "dsn_info", fmt.Sprintf("user: %s, host: %s, port: %s, dbname: %s", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName))
		return nil, fmt.Errorf("failed to connect to database: %w", err)
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
		return nil, fmt.Errorf("database is unreachable: %w", err)
	}

	// Installed for every application, single- and multi-tenant alike. The
	// callbacks return immediately for models that do not implement
	// tenant.Tenanted, so an application that opts nothing in pays nothing and
	// behaves exactly as it did before. Registering here rather than leaving it
	// to each application means a tenanted model cannot be added later and
	// silently go unscoped because someone forgot a line of wiring.
	if err := tenant.Register(db); err != nil {
		return nil, fmt.Errorf("failed to register tenant callbacks: %w", err)
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
