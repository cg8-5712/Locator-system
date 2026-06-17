package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"locator/internal/model"
)

type DeviceRepository struct {
	db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

func (r *DeviceRepository) List(ctx context.Context, limit int) ([]model.Device, error) {
	var devices []model.Device

	query := r.db.WithContext(ctx).Model(&model.Device{}).Order("last_online DESC NULLS LAST").Order("id DESC")
	if isSQLite(r.db) {
		query = r.db.WithContext(ctx).Model(&model.Device{}).Order("last_online IS NULL").Order("last_online DESC").Order("id DESC")
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}

	return devices, nil
}

func (r *DeviceRepository) GetByID(ctx context.Context, id uint64) (*model.Device, error) {
	var device model.Device
	if err := r.db.WithContext(ctx).First(&device, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}

		return nil, fmt.Errorf("get device %d: %w", id, err)
	}

	return &device, nil
}

func (r *DeviceRepository) GetTrack(ctx context.Context, deviceID uint64, startTime *time.Time, endTime *time.Time, limit int) ([]model.GPSRecord, error) {
	var records []model.GPSRecord

	query := r.db.WithContext(ctx).
		Model(&model.GPSRecord{}).
		Where("device_id = ?", deviceID).
		Order("gps_time ASC").
		Order("id ASC")

	if startTime != nil {
		query = query.Where("gps_time >= ?", startTime.UTC())
	}

	if endTime != nil {
		query = query.Where("gps_time <= ?", endTime.UTC())
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("get track for device %d: %w", deviceID, err)
	}

	return records, nil
}

func isSQLite(db *gorm.DB) bool {
	return db.Dialector.Name() == "sqlite"
}
