package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/datatypes"

	"locator/internal/model"
	"locator/internal/repository"
)

var ErrFenceNotFound = errors.New("fence not found")

type FencePoint struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
}

type FenceSummary struct {
	ID            uint64       `json:"id"`
	DeviceSN      string       `json:"device_sn"`
	Name          string       `json:"name"`
	Polygon       []FencePoint `json:"polygon"`
	LastInside    *bool        `json:"last_inside"`
	LastCheckedAt *time.Time   `json:"last_checked_at"`
	CreatedAt     time.Time    `json:"created_at"`
}

type FenceCreateInput struct {
	DeviceSN string
	Name     string
	Polygon  []FencePoint
}

type FenceUpdateInput struct {
	Name    string
	Polygon []FencePoint
}

type FenceService struct {
	deviceRepo *repository.DeviceRepository
	fenceRepo  *repository.FenceRepository
}

func NewFenceService(deviceRepo *repository.DeviceRepository, fenceRepo *repository.FenceRepository) *FenceService {
	return &FenceService{
		deviceRepo: deviceRepo,
		fenceRepo:  fenceRepo,
	}
}

func (s *FenceService) ListFences(ctx context.Context, deviceSN string) ([]FenceSummary, error) {
	device, err := s.loadDevice(ctx, deviceSN)
	if err != nil {
		return nil, err
	}

	fences, err := s.fenceRepo.ListByDeviceID(ctx, device.ID)
	if err != nil {
		return nil, err
	}

	result := make([]FenceSummary, 0, len(fences))
	for _, fence := range fences {
		summary, mapErr := mapFenceSummary(device.DeviceSN, fence)
		if mapErr != nil {
			return nil, mapErr
		}
		result = append(result, summary)
	}

	return result, nil
}

func (s *FenceService) GetFence(ctx context.Context, deviceSN string, fenceID uint64) (*FenceSummary, error) {
	device, err := s.loadDevice(ctx, deviceSN)
	if err != nil {
		return nil, err
	}

	fence, err := s.fenceRepo.GetByID(ctx, device.ID, fenceID)
	if err != nil {
		return nil, err
	}
	if fence == nil {
		return nil, ErrFenceNotFound
	}

	summary, err := mapFenceSummary(device.DeviceSN, *fence)
	if err != nil {
		return nil, err
	}

	return &summary, nil
}

func (s *FenceService) CreateFence(ctx context.Context, input FenceCreateInput) (*FenceSummary, error) {
	device, err := s.loadDevice(ctx, input.DeviceSN)
	if err != nil {
		return nil, err
	}

	name, polygonJSON, err := normalizeFenceInput(input.Name, input.Polygon)
	if err != nil {
		return nil, err
	}

	fence := model.Fence{
		DeviceID: device.ID,
		Name:     name,
		Polygon:  polygonJSON,
	}
	if err := s.fenceRepo.Create(ctx, &fence); err != nil {
		return nil, err
	}

	summary, err := mapFenceSummary(device.DeviceSN, fence)
	if err != nil {
		return nil, err
	}

	return &summary, nil
}

func (s *FenceService) UpdateFence(ctx context.Context, deviceSN string, fenceID uint64, input FenceUpdateInput) (*FenceSummary, error) {
	device, err := s.loadDevice(ctx, deviceSN)
	if err != nil {
		return nil, err
	}

	fence, err := s.fenceRepo.GetByID(ctx, device.ID, fenceID)
	if err != nil {
		return nil, err
	}
	if fence == nil {
		return nil, ErrFenceNotFound
	}

	name, polygonJSON, err := normalizeFenceInput(input.Name, input.Polygon)
	if err != nil {
		return nil, err
	}

	if err := s.fenceRepo.Update(ctx, fence, name, polygonJSON); err != nil {
		return nil, err
	}

	updated, err := s.fenceRepo.GetByID(ctx, device.ID, fenceID)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrFenceNotFound
	}

	summary, err := mapFenceSummary(device.DeviceSN, *updated)
	if err != nil {
		return nil, err
	}

	return &summary, nil
}

func (s *FenceService) DeleteFence(ctx context.Context, deviceSN string, fenceID uint64) error {
	device, err := s.loadDevice(ctx, deviceSN)
	if err != nil {
		return err
	}

	if err := s.fenceRepo.Delete(ctx, device.ID, fenceID); err != nil {
		if errors.Is(err, repository.ErrFenceNotFound) {
			return ErrFenceNotFound
		}

		return err
	}

	return nil
}

func (s *FenceService) loadDevice(ctx context.Context, deviceSN string) (*model.Device, error) {
	normalizedSN, err := normalizeDeviceSN(deviceSN)
	if err != nil {
		return nil, err
	}

	device, err := s.deviceRepo.GetByDeviceSN(ctx, normalizedSN)
	if err != nil {
		return nil, err
	}
	if device == nil {
		return nil, ErrDeviceNotFound
	}

	return device, nil
}

func normalizeFenceInput(name string, polygon []FencePoint) (string, datatypes.JSON, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return "", nil, errors.New("fence name is required")
	}

	if len(polygon) < 3 {
		return "", nil, errors.New("fence polygon must contain at least 3 points")
	}

	encodedPolygon := make([][]float64, 0, len(polygon))
	for _, point := range polygon {
		if point.Latitude < -90 || point.Latitude > 90 {
			return "", nil, errors.New("fence polygon latitude must be between -90 and 90")
		}
		if point.Longitude < -180 || point.Longitude > 180 {
			return "", nil, errors.New("fence polygon longitude must be between -180 and 180")
		}

		encodedPolygon = append(encodedPolygon, []float64{point.Latitude, point.Longitude})
	}

	raw, err := json.Marshal(encodedPolygon)
	if err != nil {
		return "", nil, err
	}

	return trimmedName, datatypes.JSON(raw), nil
}

func mapFenceSummary(deviceSN string, fence model.Fence) (FenceSummary, error) {
	var rawPolygon [][]float64
	if err := json.Unmarshal(fence.Polygon, &rawPolygon); err != nil {
		return FenceSummary{}, err
	}

	polygon := make([]FencePoint, 0, len(rawPolygon))
	for _, pair := range rawPolygon {
		if len(pair) < 2 {
			continue
		}

		polygon = append(polygon, FencePoint{
			Latitude:  pair[0],
			Longitude: pair[1],
		})
	}

	return FenceSummary{
		ID:            fence.ID,
		DeviceSN:      deviceSN,
		Name:          fence.Name,
		Polygon:       polygon,
		LastInside:    fence.LastInside,
		LastCheckedAt: fence.LastCheckedAt,
		CreatedAt:     fence.CreatedAt,
	}, nil
}
