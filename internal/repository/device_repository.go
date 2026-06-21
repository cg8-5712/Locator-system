package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"locator/internal/model"
)

var (
	ErrDeviceSNConflict = errors.New("device_sn already exists")
	ErrIMEIConflict     = errors.New("imei is already bound to another device")
	ErrDeviceNotFound   = errors.New("device not found")
)

type DeviceListFilter struct {
	DeviceSN string
	IMEI     string
	ICCID    string
	Name     string
	Status   *int
	Page     int
	PageSize int
}

type TrackPageFilter struct {
	StartTime *time.Time
	EndTime   *time.Time
	Page      int
	PageSize  int
}

type DeviceRepository struct {
	db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

func (r *DeviceRepository) List(ctx context.Context, filter DeviceListFilter) ([]model.Device, int64, error) {
	var (
		devices []model.Device
		total   int64
	)

	query := r.db.WithContext(ctx).Model(&model.Device{})
	query = applyDeviceListFilters(query, filter)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count devices: %w", err)
	}

	query = applyDeviceOrder(query, r.db)
	if filter.PageSize > 0 {
		offset := 0
		if filter.Page > 1 {
			offset = (filter.Page - 1) * filter.PageSize
		}

		query = query.Offset(offset).Limit(filter.PageSize)
	}

	if err := query.Find(&devices).Error; err != nil {
		return nil, 0, fmt.Errorf("list devices: %w", err)
	}

	return devices, total, nil
}

func (r *DeviceRepository) GetByDeviceSN(ctx context.Context, deviceSN string) (*model.Device, error) {
	var device model.Device
	if err := r.db.WithContext(ctx).Where("device_sn = ?", deviceSN).Take(&device).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("get device %s: %w", deviceSN, err)
	}

	return &device, nil
}

func (r *DeviceRepository) Create(ctx context.Context, device *model.Device) error {
	if err := r.db.WithContext(ctx).Create(device).Error; err != nil {
		return translateWriteError(err)
	}

	return nil
}

func (r *DeviceRepository) UpdateByDeviceSN(ctx context.Context, deviceSN string, updates map[string]any) (*model.Device, error) {
	device, err := r.GetByDeviceSN(ctx, deviceSN)
	if err != nil {
		return nil, err
	}

	if device == nil {
		return nil, ErrDeviceNotFound
	}

	if err := r.db.WithContext(ctx).Model(&model.Device{}).Where("device_sn = ?", deviceSN).Updates(updates).Error; err != nil {
		return nil, translateWriteError(err)
	}

	return r.GetByDeviceSN(ctx, deviceSN)
}

func (r *DeviceRepository) DeleteByDeviceSN(ctx context.Context, deviceSN string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var device model.Device
		if err := tx.Where("device_sn = ?", deviceSN).Take(&device).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrDeviceNotFound
			}

			return fmt.Errorf("load device %s for delete: %w", deviceSN, err)
		}

		if err := tx.Where("device_id = ?", device.ID).Delete(&model.GPSRecord{}).Error; err != nil {
			return fmt.Errorf("delete gps records for device %s: %w", deviceSN, err)
		}

		if err := tx.Where("device_id = ?", device.ID).Delete(&model.Fence{}).Error; err != nil {
			return fmt.Errorf("delete fences for device %s: %w", deviceSN, err)
		}

		if err := tx.Where("device_id = ?", device.ID).Delete(&model.Alarm{}).Error; err != nil {
			return fmt.Errorf("delete alarms for device %s: %w", deviceSN, err)
		}

		if err := tx.Delete(&model.Device{}, "id = ?", device.ID).Error; err != nil {
			return fmt.Errorf("delete device %s: %w", deviceSN, err)
		}

		return nil
	})
}

func (r *DeviceRepository) GetTrackByDeviceID(ctx context.Context, deviceID uint64, filter TrackPageFilter) ([]model.GPSRecord, int64, error) {
	var (
		records []model.GPSRecord
		total   int64
	)

	query := r.db.WithContext(ctx).
		Model(&model.GPSRecord{}).
		Where("device_id = ?", deviceID)

	if filter.StartTime != nil {
		query = query.Where("gps_time >= ?", filter.StartTime.UTC())
	}

	if filter.EndTime != nil {
		query = query.Where("gps_time <= ?", filter.EndTime.UTC())
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count track for device %d: %w", deviceID, err)
	}

	query = query.Order("gps_time ASC").Order("id ASC")
	if filter.PageSize > 0 {
		offset := 0
		if filter.Page > 1 {
			offset = (filter.Page - 1) * filter.PageSize
		}

		query = query.Offset(offset).Limit(filter.PageSize)
	}

	if err := query.Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("get track for device %d: %w", deviceID, err)
	}

	return records, total, nil
}

func applyDeviceListFilters(query *gorm.DB, filter DeviceListFilter) *gorm.DB {
	query = applyContainsFilter(query, "device_sn", filter.DeviceSN)
	query = applyContainsFilter(query, "imei", filter.IMEI)
	query = applyContainsFilter(query, "iccid", filter.ICCID)
	query = applyContainsFilter(query, "name", filter.Name)

	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	return query
}

func applyContainsFilter(query *gorm.DB, column string, value string) *gorm.DB {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return query
	}

	return query.Where("LOWER("+column+") LIKE ?", "%"+strings.ToLower(trimmed)+"%")
}

func applyDeviceOrder(query *gorm.DB, db *gorm.DB) *gorm.DB {
	if isSQLite(db) {
		return query.Order("last_online IS NULL").Order("last_online DESC").Order("device_sn ASC")
	}

	return query.Order("last_online DESC NULLS LAST").Order("device_sn ASC")
}

func translateWriteError(err error) error {
	if err == nil {
		return nil
	}

	lower := strings.ToLower(err.Error())
	switch {
	case strings.Contains(lower, "device_sn") && (strings.Contains(lower, "unique constraint failed") || strings.Contains(lower, "duplicate key value")):
		return fmt.Errorf("%w: %v", ErrDeviceSNConflict, err)
	case strings.Contains(lower, "imei") && (strings.Contains(lower, "unique constraint failed") || strings.Contains(lower, "duplicate key value")):
		return fmt.Errorf("%w: %v", ErrIMEIConflict, err)
	default:
		return err
	}
}

func isSQLite(db *gorm.DB) bool {
	return db.Dialector.Name() == "sqlite"
}
