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

type AlarmListFilter struct {
	DeviceSN  string
	Type      string
	StartTime *time.Time
	EndTime   *time.Time
	Page      int
	PageSize  int
}

type AlarmRow struct {
	DeviceSN  string
	Type      string
	Content   string
	CreatedAt time.Time
}

type AlarmRepository struct {
	db *gorm.DB
}

func NewAlarmRepository(db *gorm.DB) *AlarmRepository {
	return &AlarmRepository{db: db}
}

func (r *AlarmRepository) Create(ctx context.Context, alarm *model.Alarm) error {
	if err := r.db.WithContext(ctx).Create(alarm).Error; err != nil {
		return fmt.Errorf("create alarm: %w", err)
	}

	return nil
}

func (r *AlarmRepository) LatestByDeviceAndType(ctx context.Context, deviceID uint64, alarmType string) (*model.Alarm, error) {
	var alarm model.Alarm
	if err := r.db.WithContext(ctx).
		Where("device_id = ? AND type = ?", deviceID, strings.TrimSpace(alarmType)).
		Order("created_at DESC").
		Order("id DESC").
		Take(&alarm).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("get latest alarm for device %d type %s: %w", deviceID, alarmType, err)
	}

	return &alarm, nil
}

func (r *AlarmRepository) List(ctx context.Context, filter AlarmListFilter) ([]AlarmRow, int64, error) {
	var (
		rows  []AlarmRow
		total int64
	)

	query := r.db.WithContext(ctx).
		Table("alarms").
		Joins("JOIN devices ON devices.id = alarms.device_id")

	if trimmed := strings.TrimSpace(filter.DeviceSN); trimmed != "" {
		query = query.Where("devices.device_sn = ?", trimmed)
	}

	if trimmed := strings.TrimSpace(filter.Type); trimmed != "" {
		query = query.Where("alarms.type = ?", trimmed)
	}

	if filter.StartTime != nil {
		query = query.Where("alarms.created_at >= ?", filter.StartTime.UTC())
	}

	if filter.EndTime != nil {
		query = query.Where("alarms.created_at <= ?", filter.EndTime.UTC())
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count alarms: %w", err)
	}

	query = query.Select("devices.device_sn AS device_sn, alarms.type AS type, alarms.content AS content, alarms.created_at AS created_at").
		Order("alarms.created_at DESC").
		Order("alarms.id DESC")

	if filter.PageSize > 0 {
		offset := 0
		if filter.Page > 1 {
			offset = (filter.Page - 1) * filter.PageSize
		}

		query = query.Offset(offset).Limit(filter.PageSize)
	}

	if err := query.Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list alarms: %w", err)
	}

	return rows, total, nil
}
