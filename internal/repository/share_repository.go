package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"locator/internal/model"
)

var (
	ErrLocationShareNotFound = errors.New("location share not found")
)

type ShareListFilter struct {
	DeviceID  *uint64
	Page      int
	PageSize  int
}

type ShareWithDevice struct {
	Share  model.LocationShare
	Device model.Device
}

type ShareRepository struct {
	db *gorm.DB
}

func NewShareRepository(db *gorm.DB) *ShareRepository {
	return &ShareRepository{db: db}
}

func (r *ShareRepository) List(ctx context.Context, filter ShareListFilter) ([]ShareWithDevice, int64, error) {
	var (
		shares []model.LocationShare
		total  int64
	)

	query := r.db.WithContext(ctx).Model(&model.LocationShare{})
	if filter.DeviceID != nil {
		query = query.Where("device_id = ?", *filter.DeviceID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count location shares: %w", err)
	}

	query = query.Order("created_at DESC").Order("id DESC")
	if filter.PageSize > 0 {
		offset := 0
		if filter.Page > 1 {
			offset = (filter.Page - 1) * filter.PageSize
		}
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	if err := query.Find(&shares).Error; err != nil {
		return nil, 0, fmt.Errorf("list location shares: %w", err)
	}

	rows, err := r.attachDevices(ctx, shares)
	if err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r *ShareRepository) GetByID(ctx context.Context, shareID uint64) (*ShareWithDevice, error) {
	var share model.LocationShare
	if err := r.db.WithContext(ctx).Where("id = ?", shareID).Take(&share).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("get location share %d: %w", shareID, err)
	}

	return r.loadWithDevice(ctx, share)
}

func (r *ShareRepository) GetByCode(ctx context.Context, shareCode string) (*ShareWithDevice, error) {
	var share model.LocationShare
	if err := r.db.WithContext(ctx).
		Where("share_code = ?", strings.TrimSpace(shareCode)).
		Take(&share).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("get location share %s: %w", shareCode, err)
	}

	return r.loadWithDevice(ctx, share)
}

func (r *ShareRepository) Create(ctx context.Context, share *model.LocationShare) error {
	if err := r.db.WithContext(ctx).Create(share).Error; err != nil {
		return fmt.Errorf("create location share: %w", err)
	}

	return nil
}

func (r *ShareRepository) Revoke(ctx context.Context, shareID uint64, revokedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&model.LocationShare{}).
		Where("id = ?", shareID).
		Where("revoked_at IS NULL").
		Update("revoked_at", revokedAt.UTC())
	if result.Error != nil {
		return fmt.Errorf("revoke location share %d: %w", shareID, result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrLocationShareNotFound
	}

	return nil
}

func (r *ShareRepository) FindReusableSession(ctx context.Context, shareID uint64, viewerID string, now time.Time) (*model.LocationShareSession, error) {
	var session model.LocationShareSession
	if err := r.db.WithContext(ctx).
		Where("share_id = ? AND viewer_id = ?", shareID, strings.TrimSpace(viewerID)).
		Where("expires_at > ?", now.UTC()).
		Order("created_at DESC").
		Order("id DESC").
		Take(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("find reusable share session: %w", err)
	}

	return &session, nil
}

func (r *ShareRepository) CreateSession(ctx context.Context, session *model.LocationShareSession) error {
	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return fmt.Errorf("create location share session: %w", err)
	}

	return nil
}

func (r *ShareRepository) GetSessionByToken(ctx context.Context, shareID uint64, accessToken string, now time.Time) (*model.LocationShareSession, error) {
	var session model.LocationShareSession
	if err := r.db.WithContext(ctx).
		Where("share_id = ? AND access_token = ?", shareID, strings.TrimSpace(accessToken)).
		Where("expires_at > ?", now.UTC()).
		Take(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("get share session by token: %w", err)
	}

	return &session, nil
}

func (r *ShareRepository) TouchSession(ctx context.Context, sessionID uint64, touchedAt time.Time) error {
	if err := r.db.WithContext(ctx).
		Model(&model.LocationShareSession{}).
		Where("id = ?", sessionID).
		Updates(map[string]any{
			"last_seen_at":   touchedAt.UTC(),
			"last_access_at": touchedAt.UTC(),
		}).Error; err != nil {
		return fmt.Errorf("touch share session %d: %w", sessionID, err)
	}

	return nil
}

func (r *ShareRepository) DB() *gorm.DB {
	return r.db
}

func (r *ShareRepository) attachDevices(ctx context.Context, shares []model.LocationShare) ([]ShareWithDevice, error) {
	if len(shares) == 0 {
		return []ShareWithDevice{}, nil
	}

	deviceIDs := make([]uint64, 0, len(shares))
	for _, share := range shares {
		deviceIDs = append(deviceIDs, share.DeviceID)
	}

	var devices []model.Device
	if err := r.db.WithContext(ctx).
		Where("id IN ?", deviceIDs).
		Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("load devices for shares: %w", err)
	}

	deviceByID := make(map[uint64]model.Device, len(devices))
	for _, device := range devices {
		deviceByID[device.ID] = device
	}

	result := make([]ShareWithDevice, 0, len(shares))
	for _, share := range shares {
		device, ok := deviceByID[share.DeviceID]
		if !ok {
			continue
		}

		result = append(result, ShareWithDevice{
			Share:  share,
			Device: device,
		})
	}

	return result, nil
}

func (r *ShareRepository) loadWithDevice(ctx context.Context, share model.LocationShare) (*ShareWithDevice, error) {
	var device model.Device
	if err := r.db.WithContext(ctx).Where("id = ?", share.DeviceID).Take(&device).Error; err != nil {
		return nil, fmt.Errorf("load share device %d: %w", share.DeviceID, err)
	}

	return &ShareWithDevice{
		Share:  share,
		Device: device,
	}, nil
}
