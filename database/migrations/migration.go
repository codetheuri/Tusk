package migrations

import "gorm.io/gorm"

type Migration interface {
	// Up applies the migration.
	Up(tx *gorm.DB) error

	// Down reverts the migration.
	Down(tx *gorm.DB) error

	// Version returns the version of the migration.
	Version() string
	// Name returns the name of the migration.
	Name() string
}

//table model that tracks applied migrations
type SchemaMigrationModel struct {
	Version string `gorm:"primaryKey;type:varchar(255)"`
	Name    string `gorm:"type:varchar(255)"`
	AppliedAt int64  `gorm:"autoCreateTime"` 
}

func (SchemaMigrationModel) TableName() string{
	return "schema_migrations"
}

// append new migrayion instance to slice
var RegisteredMigrations []Migration
