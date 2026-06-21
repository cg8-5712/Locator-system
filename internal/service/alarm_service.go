package service

import (
	"context"
	"strings"
	"time"

	"locator/internal/repository"
)

type AlarmListQuery struct {
	DeviceSN  string
	Type      string
	StartTime *time.Time
	EndTime   *time.Time
	Page      int
	PageSize  int
}

type AlarmSummary struct {
	DeviceSN  string    `json:"device_sn"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type AlarmListResult struct {
	Alarms     []AlarmSummary `json:"alarms"`
	Pagination Pagination     `json:"pagination"`
}

type AlarmService struct {
	repo *repository.AlarmRepository
}

func NewAlarmService(repo *repository.AlarmRepository) *AlarmService {
	return &AlarmService{repo: repo}
}

func (s *AlarmService) ListAlarms(ctx context.Context, query AlarmListQuery) (*AlarmListResult, error) {
	if err := validateTimeRange(query.StartTime, query.EndTime); err != nil {
		return nil, err
	}

	page := normalizePage(query.Page)
	pageSize := normalizePageSize(query.PageSize, 20, 200)

	rows, total, err := s.repo.List(ctx, repository.AlarmListFilter{
		DeviceSN:  strings.TrimSpace(query.DeviceSN),
		Type:      strings.TrimSpace(query.Type),
		StartTime: query.StartTime,
		EndTime:   query.EndTime,
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		return nil, err
	}

	alarms := make([]AlarmSummary, 0, len(rows))
	for _, row := range rows {
		alarms = append(alarms, AlarmSummary{
			DeviceSN:  row.DeviceSN,
			Type:      row.Type,
			Content:   row.Content,
			CreatedAt: row.CreatedAt,
		})
	}

	return &AlarmListResult{
		Alarms:     alarms,
		Pagination: buildPagination(page, pageSize, total),
	}, nil
}
