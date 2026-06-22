package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"locator/internal/model"
)

var ErrFenceNotFound = errors.New("fence not found")

type FenceRepository struct {
	db *gorm.DB
}

func NewFenceRepository(db *gorm.DB) *FenceRepository {
	return &FenceRepository{db: db}
}

func (r *FenceRepository) ListByDeviceID(ctx context.Context, deviceID uint64) ([]model.Fence, error) {
	var fences []model.Fence
	if err := r.db.WithContext(ctx).
		Where("device_id = ?", deviceID).
		Order("id ASC").
		Find(&fences).Error; err != nil {
		return nil, fmt.Errorf("list fences for device %d: %w", deviceID, err)
	}

	return fences, nil
}

func (r *FenceRepository) GetByID(ctx context.Context, deviceID uint64, fenceID uint64) (*model.Fence, error) {
	var fence model.Fence
	if err := r.db.WithContext(ctx).
		Where("device_id = ? AND id = ?", deviceID, fenceID).
		Take(&fence).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("get fence %d for device %d: %w", fenceID, deviceID, err)
	}

	return &fence, nil
}

func (r *FenceRepository) Create(ctx context.Context, fence *model.Fence) error {
	if err := r.db.WithContext(ctx).Create(fence).Error; err != nil {
		return fmt.Errorf("create fence: %w", err)
	}

	return nil
}

func (r *FenceRepository) Update(ctx context.Context, fence *model.Fence, name string, polygon datatypes.JSON) error {
	if err := r.db.WithContext(ctx).
		Model(&model.Fence{}).
		Where("id = ? AND device_id = ?", fence.ID, fence.DeviceID).
		Updates(map[string]any{
			"name":    name,
			"polygon": polygon,
		}).Error; err != nil {
		return fmt.Errorf("update fence %d: %w", fence.ID, err)
	}

	return nil
}

func (r *FenceRepository) Delete(ctx context.Context, deviceID uint64, fenceID uint64) error {
	result := r.db.WithContext(ctx).
		Where("device_id = ? AND id = ?", deviceID, fenceID).
		Delete(&model.Fence{})
	if result.Error != nil {
		return fmt.Errorf("delete fence %d for device %d: %w", fenceID, deviceID, result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrFenceNotFound
	}

	return nil
}

func (r *FenceRepository) UpdateFenceState(ctx context.Context, fenceID uint64, inside bool, checkedAt time.Time) error {
	if err := r.db.WithContext(ctx).
		Model(&model.Fence{}).
		Where("id = ?", fenceID).
		Updates(map[string]any{
			"last_inside":     inside,
			"last_checked_at": checkedAt.UTC(),
		}).Error; err != nil {
		return fmt.Errorf("update fence state %d: %w", fenceID, err)
	}

	return nil
}
