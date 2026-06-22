package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"locator/internal/model"

	"gorm.io/gorm"
)

type AlarmRuleService struct {
	db     *gorm.DB
	dedupe time.Duration
}

func NewAlarmRuleService(db *gorm.DB, dedupeWindow time.Duration) *AlarmRuleService {
	return &AlarmRuleService{
		db:     db,
		dedupe: dedupeWindow,
	}
}

func (s *AlarmRuleService) CreateDeviceAlarm(ctx context.Context, tx *gorm.DB, deviceID uint64, deviceSN string, alarmType string, content string, createdAt time.Time) (*model.Alarm, bool, error) {
	if s == nil || s.db == nil {
		return nil, false, fmt.Errorf("alarm rule service is not initialized")
	}

	normalizedType := strings.TrimSpace(alarmType)
	if normalizedType == "" {
		return nil, false, fmt.Errorf("alarm type is required")
	}

	normalizedContent := strings.TrimSpace(content)
	if normalizedContent == "" {
		normalizedContent = normalizedType
	}

	timestamp := createdAt.UTC()
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	db := s.db.WithContext(ctx)
	if tx != nil {
		db = tx.WithContext(ctx)
	}

	if s.dedupe > 0 {
		var latest model.Alarm
		err := db.
			Where("device_id = ? AND type = ?", deviceID, normalizedType).
			Order("created_at DESC").
			Order("id DESC").
			Take(&latest).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, false, fmt.Errorf("get latest alarm for device %d type %s: %w", deviceID, normalizedType, err)
		}

		if err == nil && timestamp.Sub(latest.CreatedAt.UTC()) < s.dedupe {
			return &latest, false, nil
		}
	}

	alarm := &model.Alarm{
		DeviceID:  deviceID,
		Type:      normalizedType,
		Content:   normalizedContent,
		CreatedAt: timestamp,
	}
	if err := db.Create(alarm).Error; err != nil {
		return nil, false, fmt.Errorf("create alarm: %w", err)
	}

	return alarm, true, nil
}
