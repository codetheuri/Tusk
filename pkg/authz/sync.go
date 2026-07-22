package authz

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// SyncResult summarizes the changes performed during permission synchronization.
type SyncResult struct {
	Inserted int
	Updated  int
	Pruned   int
}

// Synchronizer handles syncing code permissions into the database.
type Synchronizer struct {
	db       *gorm.DB
	registry *Registry
}

// NewSynchronizer constructs a new permission synchronizer.
func NewSynchronizer(db *gorm.DB, registry *Registry) *Synchronizer {
	return &Synchronizer{
		db:       db,
		registry: registry,
	}
}

// Sync performs an idempotent synchronization between code permissions and database records.
func (s *Synchronizer) Sync(ctx context.Context, prune bool) (*SyncResult, error) {
	result := &SyncResult{}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		codePerms := s.registry.All()
		codePermMap := make(map[string]Permission, len(codePerms))

		for _, p := range codePerms {
			codePermMap[p.Name] = p

			var existing PermissionRecord
			err := tx.Where("name = ?", p.Name).First(&existing).Error
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					// 1. Insert missing permission
					record := PermissionRecord{
						Name:        p.Name,
						Description: p.Description,
					}
					if err := tx.Create(&record).Error; err != nil {
						return fmt.Errorf("failed to insert permission '%s': %w", p.Name, err)
					}
					result.Inserted++
					continue
				}
				return fmt.Errorf("failed to query permission '%s': %w", p.Name, err)
			}

			// 2. Update description if changed
			if existing.Description != p.Description {
				if err := tx.Model(&existing).Update("description", p.Description).Error; err != nil {
					return fmt.Errorf("failed to update permission '%s': %w", p.Name, err)
				}
				result.Updated++
			}
		}

		// 3. Optional pruning of obsolete permissions
		if prune {
			var dbPerms []PermissionRecord
			if err := tx.Find(&dbPerms).Error; err != nil {
				return fmt.Errorf("failed to fetch database permissions for pruning: %w", err)
			}

			for _, dbPerm := range dbPerms {
				if _, exists := codePermMap[dbPerm.Name]; !exists {
					// Delete associated role assignments first
					if err := tx.Where("permission_name = ?", dbPerm.Name).Delete(&RolePermission{}).Error; err != nil {
						return fmt.Errorf("failed to clean up role_permissions for '%s': %w", dbPerm.Name, err)
					}
					// Delete obsolete permission record
					if err := tx.Delete(&dbPerm).Error; err != nil {
						return fmt.Errorf("failed to prune obsolete permission '%s': %w", dbPerm.Name, err)
					}
					result.Pruned++
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
