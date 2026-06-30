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

type TrackPersistConfig struct {
	MinDistanceMeters float64
	MinHeadingChange  float64
	ForceInterval     time.Duration
	ForceOnFenceAlarm bool
	ForceOnSOSAlarm   bool
}

type MQTTMessageProcessor struct {
	db            *gorm.DB
	logger        *slog.Logger
	realtime      RealtimePublisher
	alarmRules    *AlarmRuleService
	trackPersist  TrackPersistConfig
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
		trackPersist: TrackPersistConfig{
			MinDistanceMeters: 40,
			MinHeadingChange:  30,
			ForceInterval:     3 * time.Minute,
			ForceOnFenceAlarm: true,
			ForceOnSOSAlarm:   true,
		},
	}
}

func (p *MQTTMessageProcessor) SetTrackPersistConfig(cfg TrackPersistConfig) {
	if cfg.MinDistanceMeters < 0 {
		cfg.MinDistanceMeters = 0
	}
	if cfg.MinHeadingChange < 0 {
		cfg.MinHeadingChange = 0
	}
	if cfg.ForceInterval < 0 {
		cfg.ForceInterval = 0
	}

	p.trackPersist = cfg
}

func (p *MQTTMessageProcessor) SetRealtimePublisher(publisher RealtimePublisher) {
	p.realtime = publisher
}

func (p *MQTTMessageProcessor) SetAlarmRuleService(alarmRules *AlarmRuleService) {
	p.alarmRules = alarmRules
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

	var (
		locationEvent *LocationEvent
		statusEvent   *DeviceStatusEvent
		alarmEvents   []AlarmEvent
	)

	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
		updates["last_latitude"] = latitude
		updates["last_longitude"] = longitude
		updates["last_location_at"] = gpsTime.UTC()
		updates["last_still_seconds"] = 0
		if err := updateDevice(tx, device.ID, updates); err != nil {
			return err
		}

		updatedDevice, err := loadDeviceByID(tx, device.ID)
		if err != nil {
			return err
		}

		alarmEvents, err = processFenceTransitions(ctx, tx, p.alarmRules, device.ID, topic.DeviceSN, latitude, longitude, gpsTime.UTC())
		if err != nil {
			return err
		}

		statusEvent = buildDeviceStatusEvent(*updatedDevice)
		locationEvent = &LocationEvent{
			DeviceSN:     topic.DeviceSN,
			TopicPrefix:  updatedDevice.TopicPrefix,
			Latitude:     latitude,
			Longitude:    longitude,
			Time:         gpsTime.UTC(),
			StillSeconds: 0,
			GPSState:     updatedDevice.GPSState,
			Status:       updatedDevice.Status,
		}

		return nil
	})
	if err != nil {
		return err
	}

	p.publishDeviceStatusEvent(statusEvent)
	p.publishLocationEvent(locationEvent)
	p.publishAlarmEvents(alarmEvents)
	return nil
}

func (p *MQTTMessageProcessor) handleCompactLocation(ctx context.Context, topic deviceTopic, message mqtt.ReceivedMessage) error {
	raw := strings.TrimSpace(string(message.Payload))
	if raw == "" {
		return errors.New("location payload is empty")
	}

	var (
		locationEvent *LocationEvent
		statusEvent   *DeviceStatusEvent
		alarmEvents   []AlarmEvent
	)

	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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

			updates["gps_state"] = "located"
			updates["last_fix_at"] = location.GPSTime.UTC()
			updates["status"] = statusOnline
			updates["last_latitude"] = location.Latitude
			updates["last_longitude"] = location.Longitude
			updates["last_location_at"] = location.GPSTime.UTC()
			updates["last_still_seconds"] = 0

			recentPersisted, err := loadRecentGPSRecords(tx, device.ID, 2)
			if err != nil {
				return err
			}

			if err := updateDevice(tx, device.ID, updates); err != nil {
				return err
			}

			alarmEvents, err = processFenceTransitions(ctx, tx, p.alarmRules, device.ID, topic.DeviceSN, location.Latitude, location.Longitude, location.GPSTime.UTC())
			if err != nil {
				return err
			}

			forcePersist := p.shouldForcePersistOnAlarm(tx, device.ID, location.GPSTime.UTC(), alarmEvents)
			shouldPersist := forcePersist || shouldPersistCompactLocation(p.trackPersist, recentPersisted, location)

			if shouldPersist {
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
			}

			updatedDevice, err := loadDeviceByID(tx, device.ID)
			if err != nil {
				return err
			}

			statusEvent = buildDeviceStatusEvent(*updatedDevice)
			locationEvent = &LocationEvent{
				DeviceSN:     topic.DeviceSN,
				TopicPrefix:  updatedDevice.TopicPrefix,
				Latitude:     location.Latitude,
				Longitude:    location.Longitude,
				Time:         location.GPSTime.UTC(),
				StillSeconds: 0,
				GPSState:     updatedDevice.GPSState,
				Status:       updatedDevice.Status,
			}

			return nil

		case strings.HasPrefix(raw, "S:"):
			stillSeconds, parseErr := parseStillSeconds(strings.TrimPrefix(raw, "S:"))
			if parseErr != nil {
				return parseErr
			}

			record, err := extendLastStationaryRecord(tx, device, stillSeconds)
			if err != nil {
				return err
			}

			updates["gps_state"] = "located"
			updates["status"] = statusOnline
			updates["last_still_seconds"] = stillSeconds
			if err := updateDevice(tx, device.ID, updates); err != nil {
				return err
			}

			updatedDevice, err := loadDeviceByID(tx, device.ID)
			if err != nil {
				return err
			}

			statusEvent = buildDeviceStatusEvent(*updatedDevice)
			if updatedDevice.LastLatitude != nil && updatedDevice.LastLongitude != nil && updatedDevice.LastLocationAt != nil {
				locationEvent = &LocationEvent{
					DeviceSN:     topic.DeviceSN,
					TopicPrefix:  updatedDevice.TopicPrefix,
					Latitude:     *updatedDevice.LastLatitude,
					Longitude:    *updatedDevice.LastLongitude,
					Time:         updatedDevice.LastLocationAt.UTC(),
					StillSeconds: stillSeconds,
					GPSState:     updatedDevice.GPSState,
					Status:       updatedDevice.Status,
				}
			} else if record != nil {
				locationEvent = &LocationEvent{
					DeviceSN:     topic.DeviceSN,
					TopicPrefix:  updatedDevice.TopicPrefix,
					Latitude:     record.Latitude,
					Longitude:    record.Longitude,
					Time:         record.GPSTime.UTC(),
					StillSeconds: stillSeconds,
					GPSState:     updatedDevice.GPSState,
					Status:       updatedDevice.Status,
				}
			}
			return nil

		case raw == "Z:0":
			updates["gps_state"] = "unable"
			updates["status"] = statusOnline
			updates["last_still_seconds"] = 0
			if err := updateDevice(tx, device.ID, updates); err != nil {
				return err
			}

			updatedDevice, err := loadDeviceByID(tx, device.ID)
			if err != nil {
				return err
			}

			statusEvent = buildDeviceStatusEvent(*updatedDevice)
			return nil

		default:
			return fmt.Errorf("unsupported compact location payload %q", raw)
		}
	})
	if err != nil {
		return err
	}

	p.publishDeviceStatusEvent(statusEvent)
	p.publishLocationEvent(locationEvent)
	p.publishAlarmEvents(alarmEvents)
	return nil
}

func (p *MQTTMessageProcessor) handleStatus(ctx context.Context, topic deviceTopic, payload map[string]any, message mqtt.ReceivedMessage) error {
	var statusEvent *DeviceStatusEvent

	statusPayload, err := encodePayloadJSON(payload)
	if err != nil {
		return fmt.Errorf("encode status payload: %w", err)
	}

	err = p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		device, err := findOrCreateDevice(tx, topic)
		if err != nil {
			return err
		}

		updates := buildDeviceUpdates(payload, topic, message.ReceivedAt)
		updates["status_payload"] = statusPayload
		updates["status_updated_at"] = message.ReceivedAt.UTC()
		if gpsState, ok := lookupString(payload, "gps"); ok {
			updates["gps_state"] = gpsState
		}
		if gpsState, ok := lookupString(payload, "gps"); ok && gpsState == "located" {
			if fixAgeMS, ok := lookupInt(payload, "fix_age_ms"); ok && fixAgeMS >= 0 {
				updates["last_fix_at"] = message.ReceivedAt.UTC().Add(-time.Duration(fixAgeMS) * time.Millisecond)
			}
		}

		if startup, ok := lookupInt(payload, "startup"); ok {
			if startup > 0 {
				updates["status"] = statusOnline
			}
		}

		if err := updateDevice(tx, device.ID, updates); err != nil {
			return err
		}

		updatedDevice, err := loadDeviceByID(tx, device.ID)
		if err != nil {
			return err
		}

		statusEvent = buildDeviceStatusEvent(*updatedDevice)
		return nil
	})
	if err != nil {
		return err
	}

	p.publishDeviceStatusEvent(statusEvent)
	return nil
}

func (p *MQTTMessageProcessor) handleAlarm(ctx context.Context, topic deviceTopic, payload map[string]any, message mqtt.ReceivedMessage) error {
	var (
		statusEvent *DeviceStatusEvent
		alarmEvents []AlarmEvent
	)

	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
		alarm, created, err := p.createDeviceAlarm(ctx, tx, device.ID, topic.DeviceSN, alarmType, content, alarmTime.UTC())
		if err != nil {
			return err
		}

		if err := updateDevice(tx, device.ID, buildDeviceUpdates(payload, topic, message.ReceivedAt)); err != nil {
			return err
		}

		updatedDevice, err := loadDeviceByID(tx, device.ID)
		if err != nil {
			return err
		}

		statusEvent = buildDeviceStatusEvent(*updatedDevice)
		if created {
			alarmEvents = append(alarmEvents, buildAlarmEvent(topic.DeviceSN, *alarm))
		}

		return nil
	})
	if err != nil {
		return err
	}

	p.publishDeviceStatusEvent(statusEvent)
	p.publishAlarmEvents(alarmEvents)
	return nil
}

func (p *MQTTMessageProcessor) handleConfig(ctx context.Context, topic deviceTopic, payload map[string]any, message mqtt.ReceivedMessage) error {
	var statusEvent *DeviceStatusEvent

	configPayload, err := encodePayloadJSON(payload)
	if err != nil {
		return fmt.Errorf("encode config payload: %w", err)
	}

	err = p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		device, err := findOrCreateDevice(tx, topic)
		if err != nil {
			return err
		}

		updates := buildDeviceUpdates(payload, topic, message.ReceivedAt)
		updates["status"] = statusOnline
		updates["config_payload"] = configPayload
		updates["config_updated_at"] = message.ReceivedAt.UTC()
		if err := updateDevice(tx, device.ID, updates); err != nil {
			return err
		}

		updatedDevice, err := loadDeviceByID(tx, device.ID)
		if err != nil {
			return err
		}

		statusEvent = buildDeviceStatusEvent(*updatedDevice)
		return nil
	})
	if err != nil {
		return err
	}

	p.publishDeviceStatusEvent(statusEvent)
	return nil
}

func (p *MQTTMessageProcessor) handleTest(ctx context.Context, topic deviceTopic, message mqtt.ReceivedMessage) error {
	var statusEvent *DeviceStatusEvent

	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		device, err := findOrCreateDevice(tx, topic)
		if err != nil {
			return err
		}

		if err := updateDevice(tx, device.ID, map[string]any{
			"last_online":  message.ReceivedAt.UTC(),
			"topic_prefix": topic.Prefix,
			"status":       statusOnline,
		}); err != nil {
			return err
		}

		updatedDevice, err := loadDeviceByID(tx, device.ID)
		if err != nil {
			return err
		}

		statusEvent = buildDeviceStatusEvent(*updatedDevice)
		return nil
	})
	if err != nil {
		return err
	}

	p.publishDeviceStatusEvent(statusEvent)
	return nil
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

func encodePayloadJSON(payload map[string]any) (datatypes.JSON, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return datatypes.JSON(body), nil
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

func loadLatestGPSRecord(tx *gorm.DB, deviceID uint64) (*model.GPSRecord, error) {
	records, err := loadRecentGPSRecords(tx, deviceID, 1)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	return &records[0], nil
}

func loadRecentGPSRecords(tx *gorm.DB, deviceID uint64, limit int) ([]model.GPSRecord, error) {
	if limit <= 0 {
		return nil, nil
	}

	var records []model.GPSRecord
	if err := tx.
		Where("device_id = ?", deviceID).
		Order("gps_time DESC").
		Order("id DESC").
		Limit(limit).
		Find(&records).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("load recent gps records for device %d: %w", deviceID, err)
	}

	return records, nil
}

func extendLastStationaryRecord(tx *gorm.DB, device *model.Device, stillSeconds int) (*model.GPSRecord, error) {
	record, err := loadLatestGPSRecord(tx, device.ID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		if device.LastLatitude == nil || device.LastLongitude == nil || device.LastLocationAt == nil {
			return nil, nil
		}

		record = &model.GPSRecord{
			DeviceID:     device.ID,
			Latitude:     *device.LastLatitude,
			Longitude:    *device.LastLongitude,
			GPSTime:      device.LastLocationAt.UTC(),
			StillSeconds: stillSeconds,
		}
		if err := tx.Create(record).Error; err != nil {
			return nil, fmt.Errorf("create stationary anchor record for device %d: %w", device.ID, err)
		}

		return record, nil
	}

	if err := tx.Model(&model.GPSRecord{}).
		Where("id = ?", record.ID).
		Update("still_seconds", stillSeconds).Error; err != nil {
		return nil, fmt.Errorf("update still_seconds for record %d: %w", record.ID, err)
	}

	record.StillSeconds = stillSeconds
	return record, nil
}

func shouldPersistCompactLocation(cfg TrackPersistConfig, recent []model.GPSRecord, current fullLocationPayload) bool {
	if len(recent) == 0 {
		return true
	}

	latest := recent[0]
	distanceMeters := geo.DistanceMeters(latest.Latitude, latest.Longitude, current.Latitude, current.Longitude)
	if cfg.MinDistanceMeters <= 0 || distanceMeters >= cfg.MinDistanceMeters {
		return true
	}

	if cfg.ForceInterval <= 0 || current.GPSTime.UTC().Sub(latest.GPSTime.UTC()) >= cfg.ForceInterval {
		return true
	}

	if cfg.MinHeadingChange <= 0 || len(recent) < 2 {
		return false
	}

	previous := recent[1]
	previousBearingDistance := geo.DistanceMeters(previous.Latitude, previous.Longitude, latest.Latitude, latest.Longitude)
	if previousBearingDistance < 1 {
		return false
	}

	previousBearing := geo.BearingDegrees(previous.Latitude, previous.Longitude, latest.Latitude, latest.Longitude)
	currentBearing := geo.BearingDegrees(latest.Latitude, latest.Longitude, current.Latitude, current.Longitude)
	headingDelta := smallestAngleDelta(previousBearing, currentBearing)
	return headingDelta >= cfg.MinHeadingChange
}

func smallestAngleDelta(a float64, b float64) float64 {
	delta := math.Mod(math.Abs(a-b), 360)
	if delta > 180 {
		return 360 - delta
	}

	return delta
}

func (p *MQTTMessageProcessor) shouldForcePersistOnAlarm(tx *gorm.DB, deviceID uint64, at time.Time, alarmEvents []AlarmEvent) bool {
	if len(alarmEvents) > 0 {
		for _, event := range alarmEvents {
			if p.trackPersist.ForceOnFenceAlarm && event.Type == "out_of_fence" {
				return true
			}
			if p.trackPersist.ForceOnSOSAlarm && event.Type == "sos" {
				return true
			}
		}
	}
	return false
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

func processFenceTransitions(ctx context.Context, tx *gorm.DB, alarmRules *AlarmRuleService, deviceID uint64, deviceSN string, latitude float64, longitude float64, checkedAt time.Time) ([]AlarmEvent, error) {
	var fences []model.Fence
	if err := tx.Where("device_id = ?", deviceID).Find(&fences).Error; err != nil {
		return nil, fmt.Errorf("list fences for device %s: %w", deviceSN, err)
	}

	events := make([]AlarmEvent, 0)
	for _, fence := range fences {
		polygon, err := decodeFencePolygon(fence.Polygon)
		if err != nil {
			return nil, fmt.Errorf("decode fence %d polygon: %w", fence.ID, err)
		}

		inside := geo.ContainsPoint(latitude, longitude, polygon)
		if err := tx.Model(&model.Fence{}).
			Where("id = ?", fence.ID).
			Updates(map[string]any{
				"last_inside":     inside,
				"last_checked_at": checkedAt.UTC(),
			}).Error; err != nil {
			return nil, fmt.Errorf("update fence %d state: %w", fence.ID, err)
		}

		if fence.LastInside != nil && *fence.LastInside == inside {
			continue
		}

		if fence.LastInside != nil && *fence.LastInside && !inside {
			content := fmt.Sprintf("device %s left fence %s", deviceSN, fence.Name)
			var (
				alarm   *model.Alarm
				created bool
			)
			if alarmRules != nil {
				alarm, created, err = alarmRules.CreateDeviceAlarm(ctx, tx, deviceID, deviceSN, "out_of_fence", content, checkedAt.UTC())
			} else {
				alarm = &model.Alarm{
					DeviceID:  deviceID,
					Type:      "out_of_fence",
					Content:   content,
					CreatedAt: checkedAt.UTC(),
				}
				if err = tx.Create(alarm).Error; err == nil {
					created = true
				}
			}
			if err != nil {
				return nil, fmt.Errorf("create out_of_fence alarm: %w", err)
			}
			if created {
				events = append(events, buildAlarmEvent(deviceSN, *alarm))
			}
		}
	}

	return events, nil
}

func (p *MQTTMessageProcessor) createDeviceAlarm(ctx context.Context, tx *gorm.DB, deviceID uint64, deviceSN string, alarmType string, content string, createdAt time.Time) (*model.Alarm, bool, error) {
	if p.alarmRules != nil {
		return p.alarmRules.CreateDeviceAlarm(ctx, tx, deviceID, deviceSN, alarmType, content, createdAt)
	}

	alarm := &model.Alarm{
		DeviceID:  deviceID,
		Type:      strings.TrimSpace(alarmType),
		Content:   strings.TrimSpace(content),
		CreatedAt: createdAt.UTC(),
	}
	if alarm.Content == "" {
		alarm.Content = alarm.Type
	}
	if err := tx.Create(alarm).Error; err != nil {
		return nil, false, fmt.Errorf("create alarm: %w", err)
	}

	return alarm, true, nil
}

func (p *MQTTMessageProcessor) publishLocationEvent(event *LocationEvent) {
	if p.realtime == nil || event == nil {
		return
	}

	p.realtime.PublishLocation(*event)
}

func (p *MQTTMessageProcessor) publishDeviceStatusEvent(event *DeviceStatusEvent) {
	if p.realtime == nil || event == nil {
		return
	}

	p.realtime.PublishDeviceStatus(*event)
}

func (p *MQTTMessageProcessor) publishAlarmEvents(events []AlarmEvent) {
	if p.realtime == nil {
		return
	}

	for _, event := range events {
		p.realtime.PublishAlarm(event)
	}
}

func loadDeviceByID(tx *gorm.DB, deviceID uint64) (*model.Device, error) {
	var device model.Device
	if err := tx.Where("id = ?", deviceID).Take(&device).Error; err != nil {
		return nil, fmt.Errorf("load device %d: %w", deviceID, err)
	}

	return &device, nil
}

func buildDeviceStatusEvent(device model.Device) *DeviceStatusEvent {
	return &DeviceStatusEvent{
		DeviceSN:        device.DeviceSN,
		TopicPrefix:     device.TopicPrefix,
		Status:          device.Status,
		GPSState:        device.GPSState,
		Battery:         device.Battery,
		IMEI:            stringValue(device.IMEI),
		ICCID:           stringValue(device.ICCID),
		StatusPayload:   append(datatypes.JSON(nil), device.StatusPayload...),
		ConfigPayload:   append(datatypes.JSON(nil), device.ConfigPayload...),
		StatusUpdatedAt: device.StatusUpdatedAt,
		ConfigUpdatedAt: device.ConfigUpdatedAt,
		LastOnline:      device.LastOnline,
		LastFixAt:       device.LastFixAt,
	}
}

func buildAlarmEvent(deviceSN string, alarm model.Alarm) AlarmEvent {
	return AlarmEvent{
		DeviceSN:  deviceSN,
		Type:      alarm.Type,
		Content:   alarm.Content,
		CreatedAt: alarm.CreatedAt,
	}
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
