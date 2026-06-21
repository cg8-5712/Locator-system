package service

import (
	"context"
	"testing"
	"time"

	"locator/internal/model"
	"locator/internal/repository"
)

func TestAlarmServiceListAlarms(t *testing.T) {
	store := openTestStore(t)
	defer closeTestStore(t, store)

	deviceA := model.Device{DeviceSN: "dev-a"}
	deviceB := model.Device{DeviceSN: "dev-b"}
	if err := store.DB().Create(&deviceA).Error; err != nil {
		t.Fatalf("create deviceA error = %v", err)
	}
	if err := store.DB().Create(&deviceB).Error; err != nil {
		t.Fatalf("create deviceB error = %v", err)
	}

	alarms := []model.Alarm{
		{DeviceID: deviceA.ID, Type: "sos", Content: "alarm-a1", CreatedAt: time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)},
		{DeviceID: deviceA.ID, Type: "low_battery", Content: "alarm-a2", CreatedAt: time.Date(2026, 6, 21, 11, 0, 0, 0, time.UTC)},
		{DeviceID: deviceB.ID, Type: "sos", Content: "alarm-b1", CreatedAt: time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)},
	}
	if err := store.DB().Create(&alarms).Error; err != nil {
		t.Fatalf("create alarms error = %v", err)
	}

	svc := NewAlarmService(repository.NewAlarmRepository(store.DB()))

	result, err := svc.ListAlarms(context.Background(), AlarmListQuery{
		DeviceSN: "dev-a",
		Page:     1,
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("ListAlarms() error = %v", err)
	}

	if len(result.Alarms) != 1 {
		t.Fatalf("len(result.Alarms) = %d, want 1", len(result.Alarms))
	}

	if result.Alarms[0].DeviceSN != "dev-a" || result.Alarms[0].Content != "alarm-a2" {
		t.Fatalf("first alarm = %+v, want latest dev-a alarm", result.Alarms[0])
	}

	if result.Pagination.Total != 2 || !result.Pagination.HasNext {
		t.Fatalf("pagination = %+v, want total=2 has_next=true", result.Pagination)
	}

	sosResult, err := svc.ListAlarms(context.Background(), AlarmListQuery{
		Type:     "sos",
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListAlarms() by type error = %v", err)
	}

	if len(sosResult.Alarms) != 2 {
		t.Fatalf("len(sosResult.Alarms) = %d, want 2", len(sosResult.Alarms))
	}
}
