package service

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"locator/internal/config"
	"locator/internal/database"
	"locator/internal/model"
	"locator/internal/mqtt"
	"locator/pkg/logger"
)

func TestMQTTMessageProcessorStoresGPSMessage(t *testing.T) {
	store := openTestStore(t)
	defer closeTestStore(t, store)

	processor := NewMQTTMessageProcessor(store.DB(), logger.New("error"))
	receivedAt := time.Date(2026, 6, 16, 3, 4, 5, 0, time.UTC)

	err := processor.HandleMessage(context.Background(), mqtt.ReceivedMessage{
		Topic:      "device/dev-001/gps",
		Payload:    []byte(`{"lat":39.90123,"lng":116.31234,"battery":86,"imei":"860000000000001","iccid":"8986000000000000001","timestamp":1750000000}`),
		QoS:        1,
		Retained:   false,
		ReceivedAt: receivedAt,
	})
	if err != nil {
		t.Fatalf("HandleMessage() error = %v", err)
	}

	var device model.Device
	if err := store.DB().Where("device_sn = ?", "dev-001").Take(&device).Error; err != nil {
		t.Fatalf("load device error = %v", err)
	}

	if device.DeviceSN != "dev-001" {
		t.Fatalf("device.DeviceSN = %q, want %q", device.DeviceSN, "dev-001")
	}

	if device.Battery != 86 {
		t.Fatalf("device.Battery = %d, want 86", device.Battery)
	}

	if device.IMEI == nil || *device.IMEI != "860000000000001" {
		t.Fatalf("device.IMEI = %v, want %q", device.IMEI, "860000000000001")
	}

	if device.ICCID == nil || *device.ICCID != "8986000000000000001" {
		t.Fatalf("device.ICCID = %v, want %q", device.ICCID, "8986000000000000001")
	}

	if device.LastOnline == nil || !device.LastOnline.Equal(receivedAt) {
		t.Fatalf("device.LastOnline = %v, want %v", device.LastOnline, receivedAt)
	}

	var record model.GPSRecord
	if err := store.DB().Where("device_id = ?", device.ID).Take(&record).Error; err != nil {
		t.Fatalf("load gps record error = %v", err)
	}

	if record.GPSTime.Unix() != 1750000000 {
		t.Fatalf("record.GPSTime = %v, want unix 1750000000", record.GPSTime)
	}

	if record.Latitude != 39.90123 || record.Longitude != 116.31234 {
		t.Fatalf("record coordinates = (%f, %f), want (39.90123, 116.31234)", record.Latitude, record.Longitude)
	}
}

func TestMQTTMessageProcessorStoresStatusAndAlarmMessages(t *testing.T) {
	store := openTestStore(t)
	defer closeTestStore(t, store)

	processor := NewMQTTMessageProcessor(store.DB(), logger.New("error"))

	statusReceivedAt := time.Date(2026, 6, 16, 8, 0, 0, 0, time.UTC)
	if err := processor.HandleMessage(context.Background(), mqtt.ReceivedMessage{
		Topic:      "device/dev-002/status",
		Payload:    []byte(`{"battery":55,"status":2,"device_name":"Truck 2","imei":"860000000000002","iccid":"8986000000000000002"}`),
		ReceivedAt: statusReceivedAt,
	}); err != nil {
		t.Fatalf("status HandleMessage() error = %v", err)
	}

	alarmReceivedAt := time.Date(2026, 6, 16, 8, 5, 0, 0, time.UTC)
	if err := processor.HandleMessage(context.Background(), mqtt.ReceivedMessage{
		Topic:      "device/dev-002/alarm",
		Payload:    []byte(`{"type":"sos","message":"panic button pressed","timestamp":"2026-06-16T08:04:30Z"}`),
		ReceivedAt: alarmReceivedAt,
	}); err != nil {
		t.Fatalf("alarm HandleMessage() error = %v", err)
	}

	var device model.Device
	if err := store.DB().Where("device_sn = ?", "dev-002").Take(&device).Error; err != nil {
		t.Fatalf("load device error = %v", err)
	}

	if device.Name != "Truck 2" {
		t.Fatalf("device.Name = %q, want %q", device.Name, "Truck 2")
	}

	if device.Status != 2 {
		t.Fatalf("device.Status = %d, want 2", device.Status)
	}

	if device.Battery != 55 {
		t.Fatalf("device.Battery = %d, want 55", device.Battery)
	}

	if device.IMEI == nil || *device.IMEI != "860000000000002" {
		t.Fatalf("device.IMEI = %v, want %q", device.IMEI, "860000000000002")
	}

	if device.ICCID == nil || *device.ICCID != "8986000000000000002" {
		t.Fatalf("device.ICCID = %v, want %q", device.ICCID, "8986000000000000002")
	}

	if device.LastOnline == nil || !device.LastOnline.Equal(alarmReceivedAt) {
		t.Fatalf("device.LastOnline = %v, want %v", device.LastOnline, alarmReceivedAt)
	}

	var alarm model.Alarm
	if err := store.DB().Where("device_id = ?", device.ID).Take(&alarm).Error; err != nil {
		t.Fatalf("load alarm error = %v", err)
	}

	if alarm.Type != "sos" {
		t.Fatalf("alarm.Type = %q, want %q", alarm.Type, "sos")
	}

	if alarm.Content != "panic button pressed" {
		t.Fatalf("alarm.Content = %q, want %q", alarm.Content, "panic button pressed")
	}

	if !alarm.CreatedAt.Equal(time.Date(2026, 6, 16, 8, 4, 30, 0, time.UTC)) {
		t.Fatalf("alarm.CreatedAt = %v, want 2026-06-16T08:04:30Z", alarm.CreatedAt)
	}
}

func TestParseDeviceTopic(t *testing.T) {
	topic, err := parseDeviceTopic("device/device-100/status")
	if err != nil {
		t.Fatalf("parseDeviceTopic() error = %v", err)
	}

	if topic.DeviceSN != "device-100" || topic.Kind != "status" {
		t.Fatalf("parseDeviceTopic() = %+v, want device_sn=device-100 kind=status", topic)
	}

	if _, err := parseDeviceTopic("device//gps"); err == nil {
		t.Fatal("parseDeviceTopic() expected error for empty device_sn")
	}

	if _, err := parseDeviceTopic("device/device-100/command"); err == nil {
		t.Fatal("parseDeviceTopic() expected error for unsupported topic kind")
	}
}

func TestMQTTMessageProcessorRejectsBindingSameIMEIToAnotherDevice(t *testing.T) {
	store := openTestStore(t)
	defer closeTestStore(t, store)

	processor := NewMQTTMessageProcessor(store.DB(), logger.New("error"))

	err := processor.HandleMessage(context.Background(), mqtt.ReceivedMessage{
		Topic:      "device/dev-a/status",
		Payload:    []byte(`{"imei":"860000000000099"}`),
		ReceivedAt: time.Date(2026, 6, 16, 9, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("first HandleMessage() error = %v", err)
	}

	err = processor.HandleMessage(context.Background(), mqtt.ReceivedMessage{
		Topic:      "device/dev-b/status",
		Payload:    []byte(`{"imei":"860000000000099"}`),
		ReceivedAt: time.Date(2026, 6, 16, 9, 1, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("second HandleMessage() expected imei binding error")
	}

	if !strings.Contains(err.Error(), "already bound") {
		t.Fatalf("second HandleMessage() error = %v, want imei binding error", err)
	}
}

func TestMQTTMessageProcessorStoresCompactLocationAndStillKeepalive(t *testing.T) {
	store := openTestStore(t)
	defer closeTestStore(t, store)

	processor := NewMQTTMessageProcessor(store.DB(), logger.New("error"))
	receivedAt := time.Date(2026, 6, 22, 9, 3, 54, 0, time.UTC)

	if err := processor.HandleMessage(context.Background(), mqtt.ReceivedMessage{
		Topic:      "locator/locator-esp32s3-001/location",
		Payload:    []byte("F:3956.20359N,11622.44467E,090353AA*4C"),
		ReceivedAt: receivedAt,
	}); err != nil {
		t.Fatalf("compact full location HandleMessage() error = %v", err)
	}

	if err := processor.HandleMessage(context.Background(), mqtt.ReceivedMessage{
		Topic:      "locator/locator-esp32s3-001/location",
		Payload:    []byte("S:600"),
		ReceivedAt: receivedAt.Add(10 * time.Minute),
	}); err != nil {
		t.Fatalf("compact still HandleMessage() error = %v", err)
	}

	var device model.Device
	if err := store.DB().Where("device_sn = ?", "locator-esp32s3-001").Take(&device).Error; err != nil {
		t.Fatalf("load device error = %v", err)
	}

	if device.TopicPrefix != "locator" {
		t.Fatalf("device.TopicPrefix = %q, want locator", device.TopicPrefix)
	}

	if device.GPSState != "located" {
		t.Fatalf("device.GPSState = %q, want located", device.GPSState)
	}

	var record model.GPSRecord
	if err := store.DB().Where("device_id = ?", device.ID).Take(&record).Error; err != nil {
		t.Fatalf("load gps record error = %v", err)
	}

	if record.StillSeconds != 600 {
		t.Fatalf("record.StillSeconds = %d, want 600", record.StillSeconds)
	}
}

func TestMQTTMessageProcessorStoresNoFixKeepaliveAndStatusConfig(t *testing.T) {
	store := openTestStore(t)
	defer closeTestStore(t, store)

	processor := NewMQTTMessageProcessor(store.DB(), logger.New("error"))
	receivedAt := time.Date(2026, 6, 22, 8, 0, 0, 0, time.UTC)

	if err := processor.HandleMessage(context.Background(), mqtt.ReceivedMessage{
		Topic:      "locator/locator-esp32s3-001/status",
		Payload:    []byte(`{"build":"Jun 20 2026 10:32:11","startup":1,"health":1,"gps":"located","net":1,"mqtt":1,"creg":1,"fix_age_ms":4200,"imei":"868478081658261","iccid":"89860412102570034386","fw":"+VERSION=CT12_V1.0.5"}`),
		ReceivedAt: receivedAt,
	}); err != nil {
		t.Fatalf("status HandleMessage() error = %v", err)
	}

	if err := processor.HandleMessage(context.Background(), mqtt.ReceivedMessage{
		Topic:      "locator/locator-esp32s3-001/config",
		Payload:    []byte(`{"pub_ms":30000,"gps_offline_ms":10000,"gps_unable_ms":30000,"move_m":30,"still_m":30,"still_confirm_ms":300000,"still_keepalive_ms":900000,"nofix_keepalive_ms":900000,"full_resync_ms":3600000,"health_ms":30000,"remote_cfg":1}`),
		ReceivedAt: receivedAt.Add(time.Minute),
	}); err != nil {
		t.Fatalf("config HandleMessage() error = %v", err)
	}

	if err := processor.HandleMessage(context.Background(), mqtt.ReceivedMessage{
		Topic:      "locator/locator-esp32s3-001/location",
		Payload:    []byte("Z:0"),
		ReceivedAt: receivedAt.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("no-fix HandleMessage() error = %v", err)
	}

	var device model.Device
	if err := store.DB().Where("device_sn = ?", "locator-esp32s3-001").Take(&device).Error; err != nil {
		t.Fatalf("load device error = %v", err)
	}

	if device.IMEI == nil || *device.IMEI != "868478081658261" {
		t.Fatalf("device.IMEI = %v, want 868478081658261", device.IMEI)
	}

	if device.ICCID == nil || *device.ICCID != "89860412102570034386" {
		t.Fatalf("device.ICCID = %v, want 89860412102570034386", device.ICCID)
	}

	if device.GPSState != "unable" {
		t.Fatalf("device.GPSState = %q, want unable", device.GPSState)
	}

	if device.StatusUpdatedAt == nil || !device.StatusUpdatedAt.Equal(receivedAt) {
		t.Fatalf("device.StatusUpdatedAt = %v, want %v", device.StatusUpdatedAt, receivedAt)
	}

	if device.ConfigUpdatedAt == nil || !device.ConfigUpdatedAt.Equal(receivedAt.Add(time.Minute)) {
		t.Fatalf("device.ConfigUpdatedAt = %v, want %v", device.ConfigUpdatedAt, receivedAt.Add(time.Minute))
	}

	if device.LastFixAt == nil || !device.LastFixAt.Equal(receivedAt.Add(-4200*time.Millisecond)) {
		t.Fatalf("device.LastFixAt = %v, want %v", device.LastFixAt, receivedAt.Add(-4200*time.Millisecond))
	}

	var statusPayload map[string]any
	if err := json.Unmarshal(device.StatusPayload, &statusPayload); err != nil {
		t.Fatalf("unmarshal device.StatusPayload error = %v", err)
	}
	if gps, ok := statusPayload["gps"].(string); !ok || gps != "located" {
		t.Fatalf("status payload gps = %v, want located", statusPayload["gps"])
	}

	var configPayload map[string]any
	if err := json.Unmarshal(device.ConfigPayload, &configPayload); err != nil {
		t.Fatalf("unmarshal device.ConfigPayload error = %v", err)
	}
	if remoteCfg, ok := configPayload["remote_cfg"].(float64); !ok || int(remoteCfg) != 1 {
		t.Fatalf("config payload remote_cfg = %v, want 1", configPayload["remote_cfg"])
	}
}

func openTestStore(t *testing.T) *database.Store {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "data", "locator.db")
	store, err := database.Open(config.DatabaseConfig{
		Driver:          "sqlite",
		DSN:             dbPath,
		AutoMigrate:     true,
		MaxIdleConns:    1,
		MaxOpenConns:    1,
		ConnMaxIdleTime: 0,
		ConnMaxLifetime: 0,
	}, logger.New("error"))
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}

	return store
}

func closeTestStore(t *testing.T, store *database.Store) {
	t.Helper()

	if err := store.Close(); err != nil {
		t.Fatalf("store.Close() error = %v", err)
	}
}
