package database

import (
	"fmt"

	"github.com/codetheuri/todolist/config"
	"github.com/codetheuri/todolist/pkg/logs"
)

func NewGoRMDB(cfg *config.Config, log logs.Logger) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser,
		cfg.DBPass,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
}
