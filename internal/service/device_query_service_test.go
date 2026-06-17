package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"locator/internal/model"
	"locator/internal/repository"
)

func TestDeviceQueryServiceListAndGetTrack(t *testing.T) {
	store := openTestStore(t)
	defer closeTestStore(t, store)

	now := time.Date(2026, 6, 17, 1, 0, 0, 0, time.UTC)
	device := model.Device{
		DeviceSN:   "dev-100",
		IMEI:       "860000000000100",
		ICCID:      "8986000000000000100",
		Name:       "Car 100",
		Status:     1,
		Battery:    67,
		LastOnline: &now,
	}
	if err := store.DB().Create(&device).Error; err != nil {
		t.Fatalf("create device error = %v", err)
	}

	records := []model.GPSRecord{
		{DeviceID: device.ID, Latitude: 39.90, Longitude: 116.30, GPSTime: now.Add(-2 * time.Minute)},
		{DeviceID: device.ID, Latitude: 39.91, Longitude: 116.31, GPSTime: now.Add(-1 * time.Minute)},
	}
	if err := store.DB().Create(&records).Error; err != nil {
		t.Fatalf("create records error = %v", err)
	}

	queryService := NewDeviceQueryService(repository.NewDeviceRepository(store.DB()))

	devices, err := queryService.ListDevices(context.Background(), 20)
	if err != nil {
		t.Fatalf("ListDevices() error = %v", err)
	}

	if len(devices) != 1 {
		t.Fatalf("len(devices) = %d, want 1", len(devices))
	}

	if devices[0].IMEI != "860000000000100" || devices[0].ICCID != "8986000000000000100" {
		t.Fatalf("device summary = %+v, want imei/iccid populated", devices[0])
	}

	gotDevice, err := queryService.GetDevice(context.Background(), device.ID)
	if err != nil {
		t.Fatalf("GetDevice() error = %v", err)
	}

	if gotDevice == nil || gotDevice.DeviceSN != "dev-100" {
		t.Fatalf("GetDevice() = %+v, want dev-100", gotDevice)
	}

	startTime := now.Add(-90 * time.Second)
	points, err := queryService.GetTrack(context.Background(), device.ID, &startTime, nil, 100)
	if err != nil {
		t.Fatalf("GetTrack() error = %v", err)
	}

	if len(points) != 1 {
		t.Fatalf("len(points) = %d, want 1", len(points))
	}

	if points[0].Latitude != 39.91 || points[0].Longitude != 116.31 {
		t.Fatalf("track point = %+v, want latest filtered point", points[0])
	}
}

func TestDeviceQueryServiceRejectsInvalidTimeRange(t *testing.T) {
	store := openTestStore(t)
	defer closeTestStore(t, store)

	queryService := NewDeviceQueryService(repository.NewDeviceRepository(store.DB()))
	start := time.Date(2026, 6, 17, 1, 0, 0, 0, time.UTC)
	end := start.Add(-time.Minute)

	_, err := queryService.GetTrack(context.Background(), 1, &start, &end, 10)
	if err == nil {
		t.Fatal("GetTrack() expected error for invalid time range")
	}
}

func TestDeviceQueryServiceGetTrackReturnsNotFoundForMissingDevice(t *testing.T) {
	store := openTestStore(t)
	defer closeTestStore(t, store)

	queryService := NewDeviceQueryService(repository.NewDeviceRepository(store.DB()))

	_, err := queryService.GetTrack(context.Background(), 999, nil, nil, 10)
	if !errors.Is(err, ErrDeviceNotFound) {
		t.Fatalf("GetTrack() error = %v, want ErrDeviceNotFound", err)
	}
}
