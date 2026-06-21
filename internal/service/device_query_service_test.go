package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"locator/internal/model"
	"locator/internal/repository"
)

func TestDeviceServiceCreateListGetUpdateAndTrack(t *testing.T) {
	store := openTestStore(t)
	defer closeTestStore(t, store)

	svc := NewDeviceService(repository.NewDeviceRepository(store.DB()))

	created, err := svc.CreateDevice(context.Background(), DeviceCreateInput{
		DeviceSN: "dev-100",
		IMEI:     stringPtr("860000000000100"),
		ICCID:    stringPtr("8986000000000000100"),
		Name:     stringPtr("Car 100"),
		Status:   intPtr(1),
		Battery:  intPtr(67),
	})
	if err != nil {
		t.Fatalf("CreateDevice() error = %v", err)
	}

	if created.DeviceSN != "dev-100" || created.IMEI != "860000000000100" {
		t.Fatalf("created device = %+v, want device_sn/imei populated", created)
	}

	secondLastOnline := time.Date(2026, 6, 17, 2, 0, 0, 0, time.UTC)
	second := model.Device{
		DeviceSN:   "dev-200",
		LastOnline: &secondLastOnline,
	}
	if err := store.DB().Create(&second).Error; err != nil {
		t.Fatalf("create second device error = %v", err)
	}

	device, err := store.DB().Where("device_sn = ?", "dev-100").Take(&model.Device{}).Rows()
	if err != nil {
		t.Fatalf("rows error = %v", err)
	}
	device.Close()

	var stored model.Device
	if err := store.DB().Where("device_sn = ?", "dev-100").Take(&stored).Error; err != nil {
		t.Fatalf("load stored device error = %v", err)
	}

	now := time.Date(2026, 6, 17, 1, 0, 0, 0, time.UTC)
	records := []model.GPSRecord{
		{DeviceID: stored.ID, Latitude: 39.90, Longitude: 116.30, GPSTime: now.Add(-2 * time.Minute)},
		{DeviceID: stored.ID, Latitude: 39.91, Longitude: 116.31, GPSTime: now.Add(-1 * time.Minute)},
	}
	if err := store.DB().Create(&records).Error; err != nil {
		t.Fatalf("create records error = %v", err)
	}

	listResult, err := svc.ListDevices(context.Background(), DeviceListQuery{
		DeviceSN: "dev",
		Page:     1,
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("ListDevices() error = %v", err)
	}

	if len(listResult.Devices) != 1 {
		t.Fatalf("len(listResult.Devices) = %d, want 1", len(listResult.Devices))
	}

	if listResult.Pagination.Total != 2 || !listResult.Pagination.HasNext {
		t.Fatalf("pagination = %+v, want total=2 has_next=true", listResult.Pagination)
	}

	gotDevice, err := svc.GetDevice(context.Background(), "dev-100")
	if err != nil {
		t.Fatalf("GetDevice() error = %v", err)
	}

	if gotDevice.DeviceSN != "dev-100" || gotDevice.ICCID != "8986000000000000100" {
		t.Fatalf("GetDevice() = %+v, want dev-100 with iccid", gotDevice)
	}

	updated, err := svc.UpdateDevice(context.Background(), "dev-100", DeviceUpdateInput{
		Name:    stringPtr("Car 100 Updated"),
		ICCID:   stringPtr(""),
		Battery: intPtr(88),
	})
	if err != nil {
		t.Fatalf("UpdateDevice() error = %v", err)
	}

	if updated.Name != "Car 100 Updated" || updated.ICCID != "" || updated.Battery != 88 {
		t.Fatalf("updated device = %+v, want updated fields", updated)
	}

	startTime := now.Add(-90 * time.Second)
	trackResult, err := svc.GetTrack(context.Background(), "dev-100", TrackQuery{
		StartTime: &startTime,
		Page:      1,
		PageSize:  1,
	})
	if err != nil {
		t.Fatalf("GetTrack() error = %v", err)
	}

	if len(trackResult.Tracks) != 1 {
		t.Fatalf("len(trackResult.Tracks) = %d, want 1", len(trackResult.Tracks))
	}

	if trackResult.Pagination.Total != 1 || trackResult.DeviceSN != "dev-100" {
		t.Fatalf("track result = %+v, want total=1 device_sn=dev-100", trackResult)
	}
}

func TestDeviceServiceRejectsInvalidTimeRange(t *testing.T) {
	store := openTestStore(t)
	defer closeTestStore(t, store)

	svc := NewDeviceService(repository.NewDeviceRepository(store.DB()))
	start := time.Date(2026, 6, 17, 1, 0, 0, 0, time.UTC)
	end := start.Add(-time.Minute)

	_, err := svc.GetTrack(context.Background(), "dev-100", TrackQuery{
		StartTime: &start,
		EndTime:   &end,
	})
	if err == nil {
		t.Fatal("GetTrack() expected error for invalid time range")
	}
}

func TestDeviceServiceReturnsNotFoundForMissingDevice(t *testing.T) {
	store := openTestStore(t)
	defer closeTestStore(t, store)

	svc := NewDeviceService(repository.NewDeviceRepository(store.DB()))

	_, err := svc.GetDevice(context.Background(), "missing-device")
	if !errors.Is(err, ErrDeviceNotFound) {
		t.Fatalf("GetDevice() error = %v, want ErrDeviceNotFound", err)
	}

	_, err = svc.GetTrack(context.Background(), "missing-device", TrackQuery{})
	if !errors.Is(err, ErrDeviceNotFound) {
		t.Fatalf("GetTrack() error = %v, want ErrDeviceNotFound", err)
	}
}

func TestDeviceServiceCreateAndUpdateConflictOnIMEI(t *testing.T) {
	store := openTestStore(t)
	defer closeTestStore(t, store)

	svc := NewDeviceService(repository.NewDeviceRepository(store.DB()))

	_, err := svc.CreateDevice(context.Background(), DeviceCreateInput{
		DeviceSN: "dev-a",
		IMEI:     stringPtr("860000000000999"),
	})
	if err != nil {
		t.Fatalf("first CreateDevice() error = %v", err)
	}

	_, err = svc.CreateDevice(context.Background(), DeviceCreateInput{
		DeviceSN: "dev-b",
		IMEI:     stringPtr("860000000000999"),
	})
	if !errors.Is(err, ErrIMEIConflict) {
		t.Fatalf("second CreateDevice() error = %v, want ErrIMEIConflict", err)
	}

	_, err = svc.UpdateDevice(context.Background(), "dev-a", DeviceUpdateInput{})
	if !errors.Is(err, ErrNoDeviceFieldChange) {
		t.Fatalf("UpdateDevice() error = %v, want ErrNoDeviceFieldChange", err)
	}
}

func TestDeviceServiceRejectsInvalidDeviceSNAndBattery(t *testing.T) {
	store := openTestStore(t)
	defer closeTestStore(t, store)

	svc := NewDeviceService(repository.NewDeviceRepository(store.DB()))

	_, err := svc.CreateDevice(context.Background(), DeviceCreateInput{
		DeviceSN: "bad/device",
	})
	if err == nil {
		t.Fatal("CreateDevice() expected invalid device_sn error")
	}

	_, err = svc.CreateDevice(context.Background(), DeviceCreateInput{
		DeviceSN: "dev-bad-battery",
		Battery:  intPtr(101),
	})
	if err == nil {
		t.Fatal("CreateDevice() expected invalid battery error")
	}
}

func TestDeviceServiceDeleteDeviceCascadesRelatedData(t *testing.T) {
	store := openTestStore(t)
	defer closeTestStore(t, store)

	svc := NewDeviceService(repository.NewDeviceRepository(store.DB()))

	created, err := svc.CreateDevice(context.Background(), DeviceCreateInput{
		DeviceSN: "dev-delete",
	})
	if err != nil {
		t.Fatalf("CreateDevice() error = %v", err)
	}

	var stored model.Device
	if err := store.DB().Where("device_sn = ?", created.DeviceSN).Take(&stored).Error; err != nil {
		t.Fatalf("load stored device error = %v", err)
	}

	fence := model.Fence{
		DeviceID: stored.ID,
		Name:     "fence-1",
		Polygon:  []byte(`[[39.9,116.3],[39.91,116.31],[39.92,116.32]]`),
	}
	alarm := model.Alarm{
		DeviceID: stored.ID,
		Type:     "sos",
		Content:  "help",
	}
	record := model.GPSRecord{
		DeviceID:  stored.ID,
		Latitude:  39.90,
		Longitude: 116.30,
		GPSTime:   time.Date(2026, 6, 21, 13, 0, 0, 0, time.UTC),
	}

	if err := store.DB().Create(&fence).Error; err != nil {
		t.Fatalf("create fence error = %v", err)
	}
	if err := store.DB().Create(&alarm).Error; err != nil {
		t.Fatalf("create alarm error = %v", err)
	}
	if err := store.DB().Create(&record).Error; err != nil {
		t.Fatalf("create gps record error = %v", err)
	}

	if err := svc.DeleteDevice(context.Background(), created.DeviceSN); err != nil {
		t.Fatalf("DeleteDevice() error = %v", err)
	}

	_, err = svc.GetDevice(context.Background(), created.DeviceSN)
	if !errors.Is(err, ErrDeviceNotFound) {
		t.Fatalf("GetDevice() after delete error = %v, want ErrDeviceNotFound", err)
	}

	var count int64
	if err := store.DB().Model(&model.GPSRecord{}).Where("device_id = ?", stored.ID).Count(&count).Error; err != nil {
		t.Fatalf("count gps records error = %v", err)
	}
	if count != 0 {
		t.Fatalf("gps record count = %d, want 0", count)
	}

	if err := store.DB().Model(&model.Fence{}).Where("device_id = ?", stored.ID).Count(&count).Error; err != nil {
		t.Fatalf("count fences error = %v", err)
	}
	if count != 0 {
		t.Fatalf("fence count = %d, want 0", count)
	}

	if err := store.DB().Model(&model.Alarm{}).Where("device_id = ?", stored.ID).Count(&count).Error; err != nil {
		t.Fatalf("count alarms error = %v", err)
	}
	if count != 0 {
		t.Fatalf("alarm count = %d, want 0", count)
	}
}

func intPtr(value int) *int {
	return &value
}

func stringPtr(value string) *string {
	return &value
}
