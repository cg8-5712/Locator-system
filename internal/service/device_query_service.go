package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"locator/internal/model"
	"locator/internal/repository"
)

var (
	ErrDeviceNotFound      = errors.New("device not found")
	ErrDeviceSNConflict    = errors.New("device_sn already exists")
	ErrIMEIConflict        = errors.New("imei is already bound to another device")
	ErrNoDeviceFieldChange = errors.New("no device fields to update")
	ErrInvalidTimeRange    = errors.New("start_time must be before end_time")
)

type DeviceService struct {
	repo *repository.DeviceRepository
}

type DeviceListQuery struct {
	DeviceSN string
	IMEI     string
	ICCID    string
	Name     string
	Status   *int
	Page     int
	PageSize int
}

type TrackQuery struct {
	StartTime *time.Time
	EndTime   *time.Time
	Page      int
	PageSize  int
}

type DeviceCreateInput struct {
	DeviceSN string
	IMEI     *string
	ICCID    *string
	Name     *string
	Status   *int
	Battery  *int
}

type DeviceUpdateInput struct {
	IMEI    *string
	ICCID   *string
	Name    *string
	Status  *int
	Battery *int
}

type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
}

type DeviceListResult struct {
	Devices    []DeviceSummary `json:"devices"`
	Pagination Pagination      `json:"pagination"`
}

type DeviceTrackResult struct {
	DeviceSN   string             `json:"device_sn"`
	Tracks     []DeviceTrackPoint `json:"tracks"`
	Pagination Pagination         `json:"pagination"`
}

type DeviceSummary struct {
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

func NewDeviceService(repo *repository.DeviceRepository) *DeviceService {
	return &DeviceService{repo: repo}
}

func (s *DeviceService) ListDevices(ctx context.Context, query DeviceListQuery) (*DeviceListResult, error) {
	page := normalizePage(query.Page)
	pageSize := normalizePageSize(query.PageSize, 20, 200)

	devices, total, err := s.repo.List(ctx, repository.DeviceListFilter{
		DeviceSN: strings.TrimSpace(query.DeviceSN),
		IMEI:     strings.TrimSpace(query.IMEI),
		ICCID:    strings.TrimSpace(query.ICCID),
		Name:     strings.TrimSpace(query.Name),
		Status:   query.Status,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}

	result := make([]DeviceSummary, 0, len(devices))
	for _, device := range devices {
		result = append(result, mapDeviceSummary(device))
	}

	return &DeviceListResult{
		Devices:    result,
		Pagination: buildPagination(page, pageSize, total),
	}, nil
}

func (s *DeviceService) GetDevice(ctx context.Context, deviceSN string) (*DeviceSummary, error) {
	normalizedSN, err := normalizeDeviceSN(deviceSN)
	if err != nil {
		return nil, err
	}

	device, err := s.repo.GetByDeviceSN(ctx, normalizedSN)
	if err != nil {
		return nil, err
	}

	if device == nil {
		return nil, ErrDeviceNotFound
	}

	summary := mapDeviceSummary(*device)
	return &summary, nil
}

func (s *DeviceService) GetTrack(ctx context.Context, deviceSN string, query TrackQuery) (*DeviceTrackResult, error) {
	normalizedSN, err := normalizeDeviceSN(deviceSN)
	if err != nil {
		return nil, err
	}

	if err := validateTimeRange(query.StartTime, query.EndTime); err != nil {
		return nil, err
	}

	device, err := s.repo.GetByDeviceSN(ctx, normalizedSN)
	if err != nil {
		return nil, err
	}

	if device == nil {
		return nil, ErrDeviceNotFound
	}

	page := normalizePage(query.Page)
	pageSize := normalizePageSize(query.PageSize, 100, 500)

	records, total, err := s.repo.GetTrackByDeviceID(ctx, device.ID, repository.TrackPageFilter{
		StartTime: query.StartTime,
		EndTime:   query.EndTime,
		Page:      page,
		PageSize:  pageSize,
	})
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

	return &DeviceTrackResult{
		DeviceSN:   normalizedSN,
		Tracks:     points,
		Pagination: buildPagination(page, pageSize, total),
	}, nil
}

func (s *DeviceService) CreateDevice(ctx context.Context, input DeviceCreateInput) (*DeviceSummary, error) {
	deviceSN, err := normalizeDeviceSN(input.DeviceSN)
	if err != nil {
		return nil, err
	}

	if err := validateBattery(input.Battery); err != nil {
		return nil, err
	}

	device := model.Device{
		DeviceSN: deviceSN,
	}

	if imei := normalizeNullableString(input.IMEI); imei != nil {
		device.IMEI = imei
	}
	if iccid := normalizeNullableString(input.ICCID); iccid != nil {
		device.ICCID = iccid
	}
	if input.Name != nil {
		device.Name = strings.TrimSpace(*input.Name)
	}
	if input.Status != nil {
		device.Status = *input.Status
	}
	if input.Battery != nil {
		device.Battery = *input.Battery
	}

	if err := s.repo.Create(ctx, &device); err != nil {
		return nil, translateRepositoryError(err)
	}

	summary := mapDeviceSummary(device)
	return &summary, nil
}

func (s *DeviceService) UpdateDevice(ctx context.Context, deviceSN string, input DeviceUpdateInput) (*DeviceSummary, error) {
	normalizedSN, err := normalizeDeviceSN(deviceSN)
	if err != nil {
		return nil, err
	}

	if err := validateBattery(input.Battery); err != nil {
		return nil, err
	}

	updates := make(map[string]any)
	if input.IMEI != nil {
		updates["imei"] = normalizeNullableString(input.IMEI)
	}
	if input.ICCID != nil {
		updates["iccid"] = normalizeNullableString(input.ICCID)
	}
	if input.Name != nil {
		updates["name"] = strings.TrimSpace(*input.Name)
	}
	if input.Status != nil {
		updates["status"] = *input.Status
	}
	if input.Battery != nil {
		updates["battery"] = *input.Battery
	}

	if len(updates) == 0 {
		return nil, ErrNoDeviceFieldChange
	}

	device, err := s.repo.UpdateByDeviceSN(ctx, normalizedSN, updates)
	if err != nil {
		return nil, translateRepositoryError(err)
	}

	summary := mapDeviceSummary(*device)
	return &summary, nil
}

func (s *DeviceService) DeleteDevice(ctx context.Context, deviceSN string) error {
	normalizedSN, err := normalizeDeviceSN(deviceSN)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteByDeviceSN(ctx, normalizedSN); err != nil {
		return translateRepositoryError(err)
	}

	return nil
}

func translateRepositoryError(err error) error {
	switch {
	case errors.Is(err, repository.ErrDeviceNotFound):
		return ErrDeviceNotFound
	case errors.Is(err, repository.ErrDeviceSNConflict):
		return ErrDeviceSNConflict
	case errors.Is(err, repository.ErrIMEIConflict):
		return ErrIMEIConflict
	default:
		return err
	}
}

func mapDeviceSummary(device model.Device) DeviceSummary {
	return DeviceSummary{
		DeviceSN:   device.DeviceSN,
		IMEI:       stringValue(device.IMEI),
		ICCID:      stringValue(device.ICCID),
		Name:       device.Name,
		Status:     device.Status,
		Battery:    device.Battery,
		LastOnline: device.LastOnline,
		CreatedAt:  device.CreatedAt,
	}
}

func buildPagination(page int, pageSize int, total int64) Pagination {
	totalPages := 0
	if total > 0 && pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	return Pagination{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    totalPages > 0 && page < totalPages,
	}
}

func normalizePage(page int) int {
	if page <= 0 {
		return 1
	}

	return page
}

func normalizePageSize(pageSize int, defaultValue int, maxValue int) int {
	if pageSize <= 0 {
		return defaultValue
	}

	if pageSize > maxValue {
		return maxValue
	}

	return pageSize
}

func normalizeDeviceSN(deviceSN string) (string, error) {
	trimmed := strings.TrimSpace(deviceSN)
	if trimmed == "" {
		return "", errors.New("device_sn is required")
	}

	if strings.Contains(trimmed, "/") {
		return "", errors.New("device_sn cannot contain '/'")
	}

	return trimmed, nil
}

func normalizeNullableString(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func validateBattery(battery *int) error {
	if battery == nil {
		return nil
	}

	if *battery < 0 || *battery > 100 {
		return errors.New("battery must be between 0 and 100")
	}

	return nil
}

func validateTimeRange(startTime *time.Time, endTime *time.Time) error {
	if startTime != nil && endTime != nil && startTime.After(*endTime) {
		return ErrInvalidTimeRange
	}

	return nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
