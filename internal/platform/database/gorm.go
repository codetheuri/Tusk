package database

import (
	"fmt"

	"github.com/codetheuri/todolist/config"
	"github.com/codetheuri/todolist/pkg/errors"
	"github.com/codetheuri/todolist/pkg/logger"
	_ "github.com/go-sql-driver/mysql" // MySQL driver
	"gorm.io/driver/mysql"             // Or postgres, sqlite, etc.
	"gorm.io/gorm"
)

func NewGoRMDB(cfg *config.Config, log logger.Logger) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser,
		cfg.DBPass,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Error("failed to connect to database", err, "dsn", err)
		return nil, errors.DatabaseError("failed tp connect to database", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Error("failed to get undelying sql.DB", err)
	}
	if err = sqlDB.Ping(); err != nil {
		log.Error("database is unreachable", err)
		return nil, errors.DatabaseError("database is unreachable", err)
	}
	log.Info("Database connected successfully ")
	return db, nil
}
