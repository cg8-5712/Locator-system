package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"locator/internal/model"
	"locator/internal/mqtt"
	"locator/pkg/geo"
)

const (
	deviceTopicPrefix  = "device"
	locatorTopicPrefix = "locator"

	messageKindGPS      = "gps"
	messageKindLocation = "location"
	messageKindStatus   = "status"
	messageKindAlarm    = "alarm"
	messageKindConfig   = "config"
	messageKindTest     = "test"

	statusOnline  = 1
	statusOffline = 0
)

type MQTTMessageProcessor struct {
	db     *gorm.DB
	logger *slog.Logger
}

type deviceTopic struct {
	Prefix   string
	DeviceSN string
	Kind     string
}

type fullLocationPayload struct {
	Latitude  float64
	Longitude float64
	GPSTime   time.Time
}

var imeiUniqueConstraintPattern = regexp.MustCompile(`(?i)(unique constraint failed: .*imei|duplicate key value.*imei)`)

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

	switch topic.Kind {
	case messageKindGPS:
		payload, decodeErr := decodePayload(message.Payload)
		if decodeErr != nil {
			return fmt.Errorf("decode %s payload for device %s: %w", topic.Kind, topic.DeviceSN, decodeErr)
		}
		err = p.handleLegacyGPS(ctx, topic, payload, message)
	case messageKindLocation:
		err = p.handleCompactLocation(ctx, topic, message)
	case messageKindStatus:
		payload, decodeErr := decodePayload(message.Payload)
		if decodeErr != nil {
			return fmt.Errorf("decode %s payload for device %s: %w", topic.Kind, topic.DeviceSN, decodeErr)
		}
		err = p.handleStatus(ctx, topic, payload, message)
	case messageKindAlarm:
		payload, decodeErr := decodePayload(message.Payload)
		if decodeErr != nil {
			return fmt.Errorf("decode %s payload for device %s: %w", topic.Kind, topic.DeviceSN, decodeErr)
		}
		err = p.handleAlarm(ctx, topic, payload, message)
	case messageKindConfig:
		payload, decodeErr := decodePayload(message.Payload)
		if decodeErr != nil {
			return fmt.Errorf("decode %s payload for device %s: %w", topic.Kind, topic.DeviceSN, decodeErr)
		}
		err = p.handleConfig(ctx, topic, payload, message)
	case messageKindTest:
		err = p.handleTest(ctx, topic, message)
	default:
		err = fmt.Errorf("unsupported message kind %q", topic.Kind)
	}

	if err != nil {
		return err
	}

	p.logger.Debug("mqtt message stored",
		"device_sn", topic.DeviceSN,
		"kind", topic.Kind,
		"prefix", topic.Prefix,
	)

	return nil
}

func (p *MQTTMessageProcessor) handleLegacyGPS(ctx context.Context, topic deviceTopic, payload map[string]any, message mqtt.ReceivedMessage) error {
	latitude, ok := lookupFloat64(payload, "lat", "latitude")
	if !ok {
		return errors.New("gps payload missing lat or latitude")
	}

	longitude, ok := lookupFloat64(payload, "lng", "lon", "longitude")
	if !ok {
		return errors.New("gps payload missing lng, lon or longitude")
	}

	gpsTime := lookupTimestamp(payload, message.ReceivedAt, "timestamp", "gps_time", "time")

	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		device, err := findOrCreateDevice(tx, topic)
		if err != nil {
			return err
		}

		record := model.GPSRecord{
			DeviceID:     device.ID,
			Latitude:     latitude,
			Longitude:    longitude,
			GPSTime:      gpsTime.UTC(),
			StillSeconds: 0,
		}

		if err := tx.Create(&record).Error; err != nil {
			return fmt.Errorf("create gps record: %w", err)
		}

		updates := buildDeviceUpdates(payload, topic, message.ReceivedAt)
		updates["gps_state"] = "located"
		updates["last_fix_at"] = gpsTime.UTC()
		if err := updateDevice(tx, device.ID, updates); err != nil {
			return err
		}

		if err := processFenceTransitions(tx, device.ID, topic.DeviceSN, latitude, longitude, gpsTime.UTC()); err != nil {
			return err
		}

		return nil
	})
}

func (p *MQTTMessageProcessor) handleCompactLocation(ctx context.Context, topic deviceTopic, message mqtt.ReceivedMessage) error {
	raw := strings.TrimSpace(string(message.Payload))
	if raw == "" {
		return errors.New("location payload is empty")
	}

	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		device, err := findOrCreateDevice(tx, topic)
		if err != nil {
			return err
		}

		updates := map[string]any{
			"last_online":  message.ReceivedAt.UTC(),
			"topic_prefix": topic.Prefix,
		}

		switch {
		case strings.HasPrefix(raw, "F:"):
			location, parseErr := parseCompactFullLocation(strings.TrimPrefix(raw, "F:"), message.ReceivedAt)
			if parseErr != nil {
				return parseErr
			}

			record := model.GPSRecord{
				DeviceID:     device.ID,
				Latitude:     location.Latitude,
				Longitude:    location.Longitude,
				GPSTime:      location.GPSTime.UTC(),
				StillSeconds: 0,
			}
			if err := tx.Create(&record).Error; err != nil {
				return fmt.Errorf("create compact gps record: %w", err)
			}

			updates["gps_state"] = "located"
			updates["last_fix_at"] = location.GPSTime.UTC()
			updates["status"] = statusOnline

			if err := updateDevice(tx, device.ID, updates); err != nil {
				return err
			}

			if err := processFenceTransitions(tx, device.ID, topic.DeviceSN, location.Latitude, location.Longitude, location.GPSTime.UTC()); err != nil {
				return err
			}

			return nil

		case strings.HasPrefix(raw, "S:"):
			stillSeconds, parseErr := parseStillSeconds(strings.TrimPrefix(raw, "S:"))
			if parseErr != nil {
				return parseErr
			}

			if err := extendLastStationaryRecord(tx, device.ID, stillSeconds); err != nil {
				return err
			}

			updates["gps_state"] = "located"
			updates["status"] = statusOnline
			return updateDevice(tx, device.ID, updates)

		case raw == "Z:0":
			updates["gps_state"] = "unable"
			updates["status"] = statusOnline
			return updateDevice(tx, device.ID, updates)

		default:
			return fmt.Errorf("unsupported compact location payload %q", raw)
		}
	})
}

func (p *MQTTMessageProcessor) handleStatus(ctx context.Context, topic deviceTopic, payload map[string]any, message mqtt.ReceivedMessage) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		device, err := findOrCreateDevice(tx, topic)
		if err != nil {
			return err
		}

		updates := buildDeviceUpdates(payload, topic, message.ReceivedAt)
		if gpsState, ok := lookupString(payload, "gps"); ok {
			updates["gps_state"] = gpsState
		}

		if startup, ok := lookupInt(payload, "startup"); ok {
			if startup > 0 {
				updates["status"] = statusOnline
			}
		}

		return updateDevice(tx, device.ID, updates)
	})
}

func (p *MQTTMessageProcessor) handleAlarm(ctx context.Context, topic deviceTopic, payload map[string]any, message mqtt.ReceivedMessage) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		device, err := findOrCreateDevice(tx, topic)
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

		return updateDevice(tx, device.ID, buildDeviceUpdates(payload, topic, message.ReceivedAt))
	})
}

func (p *MQTTMessageProcessor) handleConfig(ctx context.Context, topic deviceTopic, payload map[string]any, message mqtt.ReceivedMessage) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		device, err := findOrCreateDevice(tx, topic)
		if err != nil {
			return err
		}

		updates := buildDeviceUpdates(payload, topic, message.ReceivedAt)
		updates["status"] = statusOnline
		return updateDevice(tx, device.ID, updates)
	})
}

func (p *MQTTMessageProcessor) handleTest(ctx context.Context, topic deviceTopic, message mqtt.ReceivedMessage) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		device, err := findOrCreateDevice(tx, topic)
		if err != nil {
			return err
		}

		return updateDevice(tx, device.ID, map[string]any{
			"last_online":  message.ReceivedAt.UTC(),
			"topic_prefix": topic.Prefix,
			"status":       statusOnline,
		})
	})
}

func parseDeviceTopic(topic string) (deviceTopic, error) {
	parts := strings.Split(strings.TrimSpace(topic), "/")
	if len(parts) != 3 {
		return deviceTopic{}, fmt.Errorf("invalid topic %q: expected {prefix}/{device_sn}/{kind}", topic)
	}

	prefix := strings.ToLower(strings.TrimSpace(parts[0]))
	switch prefix {
	case deviceTopicPrefix, locatorTopicPrefix:
	default:
		return deviceTopic{}, fmt.Errorf("invalid topic %q: unsupported prefix %q", topic, parts[0])
	}

	deviceSN := strings.TrimSpace(parts[1])
	if deviceSN == "" {
		return deviceTopic{}, fmt.Errorf("invalid topic %q: empty device_sn", topic)
	}

	kind := strings.ToLower(strings.TrimSpace(parts[2]))
	switch prefix {
	case deviceTopicPrefix:
		switch kind {
		case messageKindGPS, messageKindStatus, messageKindAlarm:
		default:
			return deviceTopic{}, fmt.Errorf("invalid topic %q: unsupported kind %q", topic, parts[2])
		}
	case locatorTopicPrefix:
		switch kind {
		case messageKindLocation, messageKindStatus, messageKindConfig, messageKindTest, "cmd":
		default:
			return deviceTopic{}, fmt.Errorf("invalid topic %q: unsupported kind %q", topic, parts[2])
		}
	}

	return deviceTopic{
		Prefix:   prefix,
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

func parseCompactFullLocation(raw string, receivedAt time.Time) (fullLocationPayload, error) {
	parts := strings.Split(strings.TrimSpace(raw), ",")
	if len(parts) != 3 {
		return fullLocationPayload{}, fmt.Errorf("invalid compact location payload %q", raw)
	}

	latitude, err := parseCompactCoordinate(parts[0], true)
	if err != nil {
		return fullLocationPayload{}, fmt.Errorf("parse latitude: %w", err)
	}

	longitude, err := parseCompactCoordinate(parts[1], false)
	if err != nil {
		return fullLocationPayload{}, fmt.Errorf("parse longitude: %w", err)
	}

	gpsTime, err := parseCompactTime(parts[2], receivedAt)
	if err != nil {
		return fullLocationPayload{}, fmt.Errorf("parse gps time: %w", err)
	}

	return fullLocationPayload{
		Latitude:  latitude,
		Longitude: longitude,
		GPSTime:   gpsTime,
	}, nil
}

func parseCompactCoordinate(raw string, latitude bool) (float64, error) {
	value := strings.TrimSpace(raw)
	if len(value) < 2 {
		return 0, errors.New("coordinate too short")
	}

	hemisphere := value[len(value)-1]
	numeric := strings.TrimSpace(value[:len(value)-1])
	if _, err := strconv.ParseFloat(numeric, 64); err != nil {
		return 0, err
	}

	degreesWidth := 3
	if latitude {
		degreesWidth = 2
	}

	intPart := numeric
	if dot := strings.Index(numeric, "."); dot >= 0 {
		intPart = numeric[:dot]
	}

	if len(intPart) <= degreesWidth {
		return 0, errors.New("invalid coordinate degrees")
	}

	degreesPart := intPart[:degreesWidth]
	minutesPart := numeric[degreesWidth:]

	degrees, err := strconv.ParseFloat(degreesPart, 64)
	if err != nil {
		return 0, err
	}

	minutes, err := strconv.ParseFloat(minutesPart, 64)
	if err != nil {
		return 0, err
	}

	decimal := degrees + minutes/60.0
	switch hemisphere {
	case 'N', 'E':
	case 'S', 'W':
		decimal = -decimal
	default:
		return 0, fmt.Errorf("invalid hemisphere %q", string(hemisphere))
	}

	return decimal, nil
}

func parseCompactTime(raw string, receivedAt time.Time) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if len(value) < 6 {
		return time.Time{}, errors.New("time field too short")
	}

	hour, err := strconv.Atoi(value[0:2])
	if err != nil {
		return time.Time{}, err
	}
	minute, err := strconv.Atoi(value[2:4])
	if err != nil {
		return time.Time{}, err
	}
	second, err := strconv.Atoi(value[4:6])
	if err != nil {
		return time.Time{}, err
	}

	candidate := time.Date(receivedAt.UTC().Year(), receivedAt.UTC().Month(), receivedAt.UTC().Day(), hour, minute, second, 0, time.UTC)
	if candidate.Sub(receivedAt.UTC()) > 12*time.Hour {
		candidate = candidate.Add(-24 * time.Hour)
	} else if receivedAt.UTC().Sub(candidate) > 12*time.Hour {
		candidate = candidate.Add(24 * time.Hour)
	}

	return candidate.UTC(), nil
}

func parseStillSeconds(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, errors.New("still seconds is empty")
	}

	seconds, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	if seconds < 0 {
		return 0, errors.New("still seconds must be non-negative")
	}

	return seconds, nil
}

func extendLastStationaryRecord(tx *gorm.DB, deviceID uint64, stillSeconds int) error {
	var record model.GPSRecord
	if err := tx.
		Where("device_id = ?", deviceID).
		Order("gps_time DESC").
		Order("id DESC").
		Take(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}

		return fmt.Errorf("load last gps record for device %d: %w", deviceID, err)
	}

	if err := tx.Model(&model.GPSRecord{}).
		Where("id = ?", record.ID).
		Update("still_seconds", stillSeconds).Error; err != nil {
		return fmt.Errorf("update still_seconds for record %d: %w", record.ID, err)
	}

	return nil
}

func findOrCreateDevice(tx *gorm.DB, topic deviceTopic) (*model.Device, error) {
	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "device_sn"}},
		DoUpdates: clause.Assignments(map[string]any{
			"topic_prefix": topic.Prefix,
		}),
	}).Create(&model.Device{
		DeviceSN:    topic.DeviceSN,
		TopicPrefix: topic.Prefix,
	}).Error; err != nil {
		return nil, fmt.Errorf("ensure device %s exists: %w", topic.DeviceSN, err)
	}

	var device model.Device
	if err := tx.Where("device_sn = ?", topic.DeviceSN).Take(&device).Error; err != nil {
		return nil, fmt.Errorf("load device %s: %w", topic.DeviceSN, err)
	}

	return &device, nil
}

func updateDevice(tx *gorm.DB, deviceID uint64, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}

	if err := tx.Model(&model.Device{}).Where("id = ?", deviceID).Updates(updates).Error; err != nil {
		if imei, ok := updates["imei"].(string); ok && imei != "" && isIMEIUniqueConstraintError(err) {
			return fmt.Errorf("imei %s is already bound to another device", imei)
		}

		return fmt.Errorf("update device %d: %w", deviceID, err)
	}

	return nil
}

func buildDeviceUpdates(payload map[string]any, topic deviceTopic, receivedAt time.Time) map[string]any {
	updates := map[string]any{
		"last_online":  receivedAt.UTC(),
		"topic_prefix": topic.Prefix,
	}

	if imei, ok := lookupString(payload, "imei"); ok {
		updates["imei"] = imei
	}

	if iccid, ok := lookupString(payload, "iccid"); ok {
		updates["iccid"] = iccid
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
			updates["status"] = statusOnline
		} else {
			updates["status"] = statusOffline
		}
	} else if gpsState, ok := lookupString(payload, "gps"); ok {
		switch gpsState {
		case "located", "searching", "unable", "offline", "not_started":
			updates["status"] = statusOnline
		}
	}

	return updates
}

func processFenceTransitions(tx *gorm.DB, deviceID uint64, deviceSN string, latitude float64, longitude float64, checkedAt time.Time) error {
	var fences []model.Fence
	if err := tx.Where("device_id = ?", deviceID).Find(&fences).Error; err != nil {
		return fmt.Errorf("list fences for device %s: %w", deviceSN, err)
	}

	for _, fence := range fences {
		polygon, err := decodeFencePolygon(fence.Polygon)
		if err != nil {
			return fmt.Errorf("decode fence %d polygon: %w", fence.ID, err)
		}

		inside := geo.ContainsPoint(latitude, longitude, polygon)
		if err := tx.Model(&model.Fence{}).
			Where("id = ?", fence.ID).
			Updates(map[string]any{
				"last_inside":     inside,
				"last_checked_at": checkedAt.UTC(),
			}).Error; err != nil {
			return fmt.Errorf("update fence %d state: %w", fence.ID, err)
		}

		if fence.LastInside != nil && *fence.LastInside == inside {
			continue
		}

		if fence.LastInside != nil && *fence.LastInside && !inside {
			content := fmt.Sprintf("device %s left fence %s", deviceSN, fence.Name)
			alarm := model.Alarm{
				DeviceID:  deviceID,
				Type:      "out_of_fence",
				Content:   content,
				CreatedAt: checkedAt.UTC(),
			}
			if err := tx.Create(&alarm).Error; err != nil {
				return fmt.Errorf("create out_of_fence alarm: %w", err)
			}
		}
	}

	return nil
}

func decodeFencePolygon(raw datatypes.JSON) ([][]float64, error) {
	var polygon [][]float64
	if err := json.Unmarshal(raw, &polygon); err != nil {
		return nil, err
	}

	return polygon, nil
}

func isIMEIUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}

	return imeiUniqueConstraintPattern.MatchString(strings.ToLower(err.Error()))
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
		case json.Number:
			parsed, err := typed.Int64()
			if err == nil {
				return parsed != 0, true
			}
		case float64:
			return typed != 0, true
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
		return int(math.Round(typed)), true
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
