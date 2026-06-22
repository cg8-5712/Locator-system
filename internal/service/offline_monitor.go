package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"locator/internal/model"
	"locator/internal/repository"
)

type OfflineMonitorConfig struct {
	CheckInterval time.Duration
	OfflineAfter  time.Duration
}

type OfflineMonitor struct {
	db         *gorm.DB
	deviceRepo *repository.DeviceRepository
	alarmRules *AlarmRuleService
	realtime   RealtimePublisher
	logger     *slog.Logger
	cfg        OfflineMonitorConfig
}

func NewOfflineMonitor(db *gorm.DB, deviceRepo *repository.DeviceRepository, alarmRules *AlarmRuleService, realtime RealtimePublisher, logger *slog.Logger, cfg OfflineMonitorConfig) *OfflineMonitor {
	if logger == nil {
		logger = slog.Default()
	}

	return &OfflineMonitor{
		db:         db,
		deviceRepo: deviceRepo,
		alarmRules: alarmRules,
		realtime:   realtime,
		logger:     logger,
		cfg:        cfg,
	}
}

func (m *OfflineMonitor) Start(ctx context.Context) {
	if m == nil || m.db == nil || m.deviceRepo == nil {
		return
	}

	if m.cfg.CheckInterval <= 0 || m.cfg.OfflineAfter <= 0 {
		m.logger.Info("offline monitor disabled", "check_interval", m.cfg.CheckInterval, "offline_after", m.cfg.OfflineAfter)
		return
	}

	go m.loop(ctx)
}

func (m *OfflineMonitor) loop(ctx context.Context) {
	ticker := time.NewTicker(m.cfg.CheckInterval)
	defer ticker.Stop()

	m.logger.Info("offline monitor started", "check_interval", m.cfg.CheckInterval, "offline_after", m.cfg.OfflineAfter)
	defer m.logger.Info("offline monitor stopped")

	if err := m.RunOnce(ctx, time.Now().UTC()); err != nil {
		m.logger.Error("offline monitor run failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case tickAt := <-ticker.C:
			if err := m.RunOnce(ctx, tickAt.UTC()); err != nil {
				m.logger.Error("offline monitor run failed", "error", err)
			}
		}
	}
}

func (m *OfflineMonitor) RunOnce(ctx context.Context, now time.Time) error {
	if m == nil || m.db == nil || m.deviceRepo == nil {
		return nil
	}

	cutoff := now.UTC().Add(-m.cfg.OfflineAfter)
	candidates, err := m.deviceRepo.ListOfflineCandidates(ctx, cutoff)
	if err != nil {
		return err
	}

	for _, candidate := range candidates {
		if err := m.processCandidate(ctx, candidate, cutoff); err != nil {
			return err
		}
	}

	return nil
}

func (m *OfflineMonitor) processCandidate(ctx context.Context, candidate model.Device, cutoff time.Time) error {
	var (
		statusEvent *DeviceStatusEvent
		alarmEvent  *AlarmEvent
	)

	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		device, err := loadDeviceByID(tx, candidate.ID)
		if err != nil {
			return err
		}

		if device.Status == statusOffline || device.LastOnline == nil || !device.LastOnline.UTC().Before(cutoff.UTC()) {
			return nil
		}

		if err := tx.Model(&model.Device{}).
			Where("id = ?", device.ID).
			Updates(map[string]any{
				"status":    statusOffline,
				"gps_state": "offline",
			}).Error; err != nil {
			return fmt.Errorf("mark device %s offline: %w", device.DeviceSN, err)
		}

		updatedDevice, err := loadDeviceByID(tx, device.ID)
		if err != nil {
			return err
		}

		statusEvent = buildDeviceStatusEvent(*updatedDevice)

		content := fmt.Sprintf("device %s offline since %s", updatedDevice.DeviceSN, updatedDevice.LastOnline.UTC().Format(time.RFC3339))
		alarm, created, err := m.createOfflineAlarm(ctx, tx, *updatedDevice, content, cutoff.UTC())
		if err != nil {
			return err
		}

		if created {
			event := buildAlarmEvent(updatedDevice.DeviceSN, *alarm)
			alarmEvent = &event
		}

		return nil
	})
	if err != nil {
		return err
	}

	if statusEvent != nil && m.realtime != nil {
		m.realtime.PublishDeviceStatus(*statusEvent)
	}
	if alarmEvent != nil && m.realtime != nil {
		m.realtime.PublishAlarm(*alarmEvent)
	}

	return nil
}

func (m *OfflineMonitor) createOfflineAlarm(ctx context.Context, tx *gorm.DB, device model.Device, content string, createdAt time.Time) (*model.Alarm, bool, error) {
	if m.alarmRules != nil {
		return m.alarmRules.CreateDeviceAlarm(ctx, tx, device.ID, device.DeviceSN, "offline", content, createdAt)
	}

	alarm := &model.Alarm{
		DeviceID:  device.ID,
		Type:      "offline",
		Content:   content,
		CreatedAt: createdAt.UTC(),
	}
	if err := tx.Create(alarm).Error; err != nil {
		return nil, false, fmt.Errorf("create offline alarm: %w", err)
	}

	return alarm, true, nil
}
