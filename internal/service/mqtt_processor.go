package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"locator/internal/model"
	"locator/internal/mqtt"
)

const (
	deviceTopicPrefix = "device"
	messageKindGPS    = "gps"
	messageKindStatus = "status"
	messageKindAlarm  = "alarm"
)

type MQTTMessageProcessor struct {
	db     *gorm.DB
	logger *slog.Logger
}

type deviceTopic struct {
	DeviceSN string
	Kind     string
}

func NewMQTTMessageProcessor(db *gorm.DB, logger *slog.Logger) *MQTTMessageProcessor {
	if logger == nil {
		logger = slog.Default()
	}

	return &MQTTMessageProcessor{
		db:     db,
		logger: logger,
	}
}

func (p *MQTTMessageProcessor) HandleMessage(ctx context.Context, message mqtt.ReceivedMessage) error {
	if p.db == nil {
		return errors.New("mqtt message processor requires a database connection")
	}

	topic, err := parseDeviceTopic(message.Topic)
	if err != nil {
		return err
	}

	payload, err := decodePayload(message.Payload)
	if err != nil {
		return fmt.Errorf("decode %s payload for device %s: %w", topic.Kind, topic.DeviceSN, err)
	}

	switch topic.Kind {
	case messageKindGPS:
		err = p.handleGPS(ctx, topic, payload, message)
	case messageKindStatus:
		err = p.handleStatus(ctx, topic, payload, message)
	case messageKindAlarm:
		err = p.handleAlarm(ctx, topic, payload, message)
	default:
		err = fmt.Errorf("unsupported message kind %q", topic.Kind)
	}

	if err != nil {
		return err
	}

	p.logger.Debug("mqtt message stored",
		"device_sn", topic.DeviceSN,
		"kind", topic.Kind,
	)

	return nil
}

func (p *MQTTMessageProcessor) handleGPS(ctx context.Context, topic deviceTopic, payload map[string]any, message mqtt.ReceivedMessage) error {
	latitude, ok := lookupFloat64(payload, "lat", "latitude")
	if !ok {
		return errors.New("gps payload missing lat or latitude")
	}

	longitude, ok := lookupFloat64(payload, "lng", "lon", "longitude")
	if !ok {
		return errors.New("gps payload missing lng, lon or longitude")
	}

	speed, _ := lookupFloat64(payload, "speed")
	altitude, _ := lookupFloat64(payload, "altitude", "alt")
	gpsTime := lookupTimestamp(payload, message.ReceivedAt, "timestamp", "gps_time", "time")

	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		device, err := findOrCreateDevice(tx, topic.DeviceSN)
		if err != nil {
			return err
		}

		record := model.GPSRecord{
			DeviceID:  device.ID,
			Latitude:  latitude,
			Longitude: longitude,
			Speed:     float32(speed),
			Altitude:  float32(altitude),
			GPSTime:   gpsTime.UTC(),
		}

		if err := tx.Create(&record).Error; err != nil {
			return fmt.Errorf("create gps record: %w", err)
		}

		if err := updateDevice(tx, device.ID, buildDeviceUpdates(payload, message.ReceivedAt)); err != nil {
			return err
		}

		return nil
	})
}

func (p *MQTTMessageProcessor) handleStatus(ctx context.Context, topic deviceTopic, payload map[string]any, message mqtt.ReceivedMessage) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		device, err := findOrCreateDevice(tx, topic.DeviceSN)
		if err != nil {
			return err
		}

		if err := updateDevice(tx, device.ID, buildDeviceUpdates(payload, message.ReceivedAt)); err != nil {
			return err
		}

		return nil
	})
}

func (p *MQTTMessageProcessor) handleAlarm(ctx context.Context, topic deviceTopic, payload map[string]any, message mqtt.ReceivedMessage) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		device, err := findOrCreateDevice(tx, topic.DeviceSN)
		if err != nil {
			return err
		}

		alarmType, ok := lookupString(payload, "type", "alarm_type")
		if !ok {
			alarmType = messageKindAlarm
		}

		content, ok := lookupString(payload, "content", "message", "description")
		if !ok || strings.TrimSpace(content) == "" {
			content = string(message.Payload)
		}

		alarmTime := lookupTimestamp(payload, message.ReceivedAt, "timestamp", "time")
		alarm := model.Alarm{
			DeviceID:  device.ID,
			Type:      alarmType,
			Content:   content,
			CreatedAt: alarmTime.UTC(),
		}

		if err := tx.Create(&alarm).Error; err != nil {
			return fmt.Errorf("create alarm record: %w", err)
		}

		if err := updateDevice(tx, device.ID, buildDeviceUpdates(payload, message.ReceivedAt)); err != nil {
			return err
		}

		return nil
	})
}

func parseDeviceTopic(topic string) (deviceTopic, error) {
	parts := strings.Split(strings.TrimSpace(topic), "/")
	if len(parts) != 3 {
		return deviceTopic{}, fmt.Errorf("invalid topic %q: expected device/{device_sn}/{kind}", topic)
	}

	if parts[0] != deviceTopicPrefix {
		return deviceTopic{}, fmt.Errorf("invalid topic %q: expected prefix %q", topic, deviceTopicPrefix)
	}

	deviceSN := strings.TrimSpace(parts[1])
	if deviceSN == "" {
		return deviceTopic{}, fmt.Errorf("invalid topic %q: empty device_sn", topic)
	}

	kind := strings.ToLower(strings.TrimSpace(parts[2]))
	switch kind {
	case messageKindGPS, messageKindStatus, messageKindAlarm:
	default:
		return deviceTopic{}, fmt.Errorf("invalid topic %q: unsupported kind %q", topic, parts[2])
	}

	return deviceTopic{
		DeviceSN: deviceSN,
		Kind:     kind,
	}, nil
}

func decodePayload(payload []byte) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()

	var parsed map[string]any
	if err := decoder.Decode(&parsed); err != nil {
		return nil, err
	}

	if parsed == nil {
		return nil, errors.New("payload must be a JSON object")
	}

	return parsed, nil
}

func findOrCreateDevice(tx *gorm.DB, deviceSN string) (*model.Device, error) {
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "device_sn"}},
		DoNothing: true,
	}).Create(&model.Device{
		DeviceSN: deviceSN,
	}).Error; err != nil {
		return nil, fmt.Errorf("ensure device %s exists: %w", deviceSN, err)
	}

	var device model.Device
	if err := tx.Where("device_sn = ?", deviceSN).Take(&device).Error; err != nil {
		return nil, fmt.Errorf("load device %s: %w", deviceSN, err)
	}

	return &device, nil
}

func updateDevice(tx *gorm.DB, deviceID uint64, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}

	if err := tx.Model(&model.Device{}).Where("id = ?", deviceID).Updates(updates).Error; err != nil {
		return fmt.Errorf("update device %d: %w", deviceID, err)
	}

	return nil
}

func buildDeviceUpdates(payload map[string]any, receivedAt time.Time) map[string]any {
	updates := map[string]any{
		"last_online": receivedAt.UTC(),
	}

	if name, ok := lookupString(payload, "device_name", "name"); ok {
		updates["name"] = name
	}

	if battery, ok := lookupInt(payload, "battery"); ok {
		updates["battery"] = battery
	}

	if status, ok := lookupInt(payload, "status"); ok {
		updates["status"] = status
	} else if online, ok := lookupBool(payload, "online"); ok {
		if online {
			updates["status"] = 1
		} else {
			updates["status"] = 0
		}
	}

	return updates
}

func lookupString(payload map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}

		switch typed := value.(type) {
		case string:
			typed = strings.TrimSpace(typed)
			if typed != "" {
				return typed, true
			}
		case json.Number:
			return typed.String(), true
		}
	}

	return "", false
}

func lookupFloat64(payload map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}

		parsed, ok := coerceFloat64(value)
		if ok {
			return parsed, true
		}
	}

	return 0, false
}

func lookupInt(payload map[string]any, keys ...string) (int, bool) {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}

		parsed, ok := coerceInt(value)
		if ok {
			return parsed, true
		}
	}

	return 0, false
}

func lookupBool(payload map[string]any, keys ...string) (bool, bool) {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}

		switch typed := value.(type) {
		case bool:
			return typed, true
		case string:
			parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
			if err == nil {
				return parsed, true
			}
		}
	}

	return false, false
}

func lookupTimestamp(payload map[string]any, fallback time.Time, keys ...string) time.Time {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}

		parsed, ok := coerceTime(value)
		if ok {
			return parsed.UTC()
		}
	}

	return fallback.UTC()
}

func coerceFloat64(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		if err == nil {
			return parsed, true
		}
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err == nil {
			return parsed, true
		}
	}

	return 0, false
}

func coerceInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil {
			return int(parsed), true
		}
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed, true
		}
	}

	return 0, false
}

func coerceTime(value any) (time.Time, bool) {
	switch typed := value.(type) {
	case time.Time:
		return typed.UTC(), true
	case json.Number:
		if parsed, ok := parseUnixNumber(typed.String()); ok {
			return parsed, true
		}
	case float64:
		if parsed, ok := unixFromFloat(typed); ok {
			return parsed, true
		}
	case int64:
		return time.Unix(typed, 0).UTC(), true
	case int:
		return time.Unix(int64(typed), 0).UTC(), true
	case string:
		raw := strings.TrimSpace(typed)
		if raw == "" {
			return time.Time{}, false
		}

		if parsed, ok := parseUnixNumber(raw); ok {
			return parsed, true
		}

		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
			parsed, err := time.Parse(layout, raw)
			if err == nil {
				return parsed.UTC(), true
			}
		}
	}

	return time.Time{}, false
}

func parseUnixNumber(raw string) (time.Time, bool) {
	if raw == "" {
		return time.Time{}, false
	}

	if strings.Contains(raw, ".") {
		floatValue, err := strconv.ParseFloat(raw, 64)
		if err == nil {
			return unixFromFloat(floatValue)
		}
	}

	intValue, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, false
	}

	return unixFromInt(intValue), true
}

func unixFromFloat(value float64) (time.Time, bool) {
	seconds := int64(value)
	return unixFromInt(seconds), true
}

func unixFromInt(value int64) time.Time {
	switch {
	case value > 1_000_000_000_000:
		return time.UnixMilli(value).UTC()
	default:
		return time.Unix(value, 0).UTC()
	}
}
