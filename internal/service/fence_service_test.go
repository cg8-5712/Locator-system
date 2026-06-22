package service

import (
	"context"
	"errors"
	"testing"

	"locator/internal/repository"
)

func TestFenceServiceCreateListGetUpdateDelete(t *testing.T) {
	store := openTestStore(t)
	defer closeTestStore(t, store)

	deviceSvc := NewDeviceService(repository.NewDeviceRepository(store.DB()))
	_, err := deviceSvc.CreateDevice(context.Background(), DeviceCreateInput{
		DeviceSN: "dev-fence",
	})
	if err != nil {
		t.Fatalf("CreateDevice() error = %v", err)
	}

	svc := NewFenceService(
		repository.NewDeviceRepository(store.DB()),
		repository.NewFenceRepository(store.DB()),
	)

	created, err := svc.CreateFence(context.Background(), FenceCreateInput{
		DeviceSN: "dev-fence",
		Name:     "yard",
		Polygon: []FencePoint{
			{Latitude: 39.90, Longitude: 116.30},
			{Latitude: 39.91, Longitude: 116.31},
			{Latitude: 39.92, Longitude: 116.30},
		},
	})
	if err != nil {
		t.Fatalf("CreateFence() error = %v", err)
	}

	if created.DeviceSN != "dev-fence" || created.Name != "yard" {
		t.Fatalf("created fence = %+v", created)
	}

	list, err := svc.ListFences(context.Background(), "dev-fence")
	if err != nil {
		t.Fatalf("ListFences() error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}

	got, err := svc.GetFence(context.Background(), "dev-fence", created.ID)
	if err != nil {
		t.Fatalf("GetFence() error = %v", err)
	}
	if got.Name != "yard" {
		t.Fatalf("GetFence() = %+v", got)
	}

	updated, err := svc.UpdateFence(context.Background(), "dev-fence", created.ID, FenceUpdateInput{
		Name: "yard-updated",
		Polygon: []FencePoint{
			{Latitude: 39.90, Longitude: 116.30},
			{Latitude: 39.91, Longitude: 116.32},
			{Latitude: 39.93, Longitude: 116.31},
		},
	})
	if err != nil {
		t.Fatalf("UpdateFence() error = %v", err)
	}
	if updated.Name != "yard-updated" {
		t.Fatalf("updated fence = %+v", updated)
	}

	if err := svc.DeleteFence(context.Background(), "dev-fence", created.ID); err != nil {
		t.Fatalf("DeleteFence() error = %v", err)
	}

	_, err = svc.GetFence(context.Background(), "dev-fence", created.ID)
	if !errors.Is(err, ErrFenceNotFound) {
		t.Fatalf("GetFence() after delete error = %v, want ErrFenceNotFound", err)
	}
}

func TestFenceServiceRejectsInvalidPolygon(t *testing.T) {
	store := openTestStore(t)
	defer closeTestStore(t, store)

	deviceSvc := NewDeviceService(repository.NewDeviceRepository(store.DB()))
	_, err := deviceSvc.CreateDevice(context.Background(), DeviceCreateInput{
		DeviceSN: "dev-fence-bad",
	})
	if err != nil {
		t.Fatalf("CreateDevice() error = %v", err)
	}

	svc := NewFenceService(
		repository.NewDeviceRepository(store.DB()),
		repository.NewFenceRepository(store.DB()),
	)

	_, err = svc.CreateFence(context.Background(), FenceCreateInput{
		DeviceSN: "dev-fence-bad",
		Name:     "bad",
		Polygon: []FencePoint{
			{Latitude: 39.90, Longitude: 116.30},
			{Latitude: 39.91, Longitude: 116.31},
		},
	})
	if err == nil {
		t.Fatal("CreateFence() expected polygon validation error")
	}
}
