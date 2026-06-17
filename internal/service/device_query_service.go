package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"locator/internal/model"
	"locator/internal/repository"
)

var ErrDeviceNotFound = errors.New("device not found")

type DeviceQueryService struct {
	repo *repository.DeviceRepository
}

type DeviceSummary struct {
	ID         uint64     `json:"id"`
	DeviceSN   string     `json:"device_sn"`
	IMEI       string     `json:"imei"`
	ICCID      string     `json:"iccid"`
	Name       string     `json:"name"`
	Status     int        `json:"status"`
	Battery    int        `json:"battery"`
	LastOnline *time.Time `json:"last_online"`
	CreatedAt  time.Time  `json:"created_at"`
}

type DeviceTrackPoint struct {
	Latitude  float64   `json:"lat"`
	Longitude float64   `json:"lng"`
	Time      time.Time `json:"time"`
}

func NewDeviceQueryService(repo *repository.DeviceRepository) *DeviceQueryService {
	return &DeviceQueryService{repo: repo}
}

func (s *DeviceQueryService) ListDevices(ctx context.Context, limit int) ([]DeviceSummary, error) {
	devices, err := s.repo.List(ctx, limit)
	if err != nil {
		return nil, err
	}

	result := make([]DeviceSummary, 0, len(devices))
	for _, device := range devices {
		result = append(result, mapDeviceSummary(device))
	}

	return result, nil
}

func (s *DeviceQueryService) GetDevice(ctx context.Context, id uint64) (*DeviceSummary, error) {
	device, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if device == nil {
		return nil, nil
	}

	summary := mapDeviceSummary(*device)
	return &summary, nil
}

func (s *DeviceQueryService) GetTrack(ctx context.Context, deviceID uint64, startTime *time.Time, endTime *time.Time, limit int) ([]DeviceTrackPoint, error) {
	if startTime != nil && endTime != nil && startTime.After(*endTime) {
		return nil, fmt.Errorf("start_time must be before end_time")
	}

	device, err := s.repo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	if device == nil {
		return nil, ErrDeviceNotFound
	}

	records, err := s.repo.GetTrack(ctx, deviceID, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}

	points := make([]DeviceTrackPoint, 0, len(records))
	for _, record := range records {
		points = append(points, DeviceTrackPoint{
			Latitude:  record.Latitude,
			Longitude: record.Longitude,
			Time:      record.GPSTime,
		})
	}

	return points, nil
}

func mapDeviceSummary(device model.Device) DeviceSummary {
	return DeviceSummary{
		ID:         device.ID,
		DeviceSN:   device.DeviceSN,
		IMEI:       device.IMEI,
		ICCID:      device.ICCID,
		Name:       device.Name,
		Status:     device.Status,
		Battery:    device.Battery,
		LastOnline: device.LastOnline,
		CreatedAt:  device.CreatedAt,
	}
}
