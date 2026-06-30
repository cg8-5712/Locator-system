package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"locator/internal/model"
	"locator/internal/repository"
)

const (
	ShareModeLiveOnly   = "live_only"
	ShareModeTodayTrack = "today_track"
)

var (
	ErrShareNotFound         = errors.New("share not found")
	ErrShareExpired          = errors.New("share expired")
	ErrShareRevoked          = errors.New("share revoked")
	ErrSharePasswordRequired = errors.New("share password is required")
	ErrShareInvalidPassword  = errors.New("share password is invalid")
	ErrShareVisitLimit       = errors.New("share visit limit reached")
	ErrShareAccessDenied     = errors.New("share access denied")
	ErrShareTrackNotAllowed  = errors.New("share track access is not allowed")
	ErrInvalidShareMode      = errors.New("invalid share mode")
	ErrInvalidShareExpiry    = errors.New("share expiry must be in the future")
	ErrInvalidViewerID       = errors.New("viewer_id is required")
)

type ShareListQuery struct {
	DeviceSN string
	Page     int
	PageSize int
}

type ShareCreateInput struct {
	DeviceSN        string
	ShareMode       string
	Password        *string
	ExpiresAt       time.Time
	MaxVisits       *int
	Note            string
	CreatedByUserID *uint64
}

type ShareSummary struct {
	ID               uint64     `json:"id"`
	DeviceSN         string     `json:"device_sn"`
	DeviceName       string     `json:"device_name"`
	ShareCode        string     `json:"share_code"`
	ShareMode        string     `json:"share_mode"`
	RequiresPassword bool       `json:"requires_password"`
	Note             string     `json:"note"`
	ExpiresAt        time.Time  `json:"expires_at"`
	MaxVisits        *int       `json:"max_visits"`
	VisitCount       int        `json:"visit_count"`
	RemainingVisits  *int       `json:"remaining_visits"`
	LastAccessAt     *time.Time `json:"last_access_at"`
	RevokedAt        *time.Time `json:"revoked_at"`
	CreatedAt        time.Time  `json:"created_at"`
	Status           string     `json:"status"`
}

type ShareCreateResult struct {
	Share    ShareSummary `json:"share"`
	Password *string      `json:"password,omitempty"`
}

type ShareListResult struct {
	Shares      []ShareSummary `json:"shares"`
	Pagination  Pagination     `json:"pagination"`
}

type PublicShareSummary struct {
	ShareCode         string     `json:"share_code"`
	DeviceSN          string     `json:"device_sn"`
	DeviceName        string     `json:"device_name"`
	ShareMode         string     `json:"share_mode"`
	RequiresPassword  bool       `json:"requires_password"`
	ExpiresAt         time.Time  `json:"expires_at"`
	MaxVisits         *int       `json:"max_visits"`
	VisitCount        int        `json:"visit_count"`
	RemainingVisits   *int       `json:"remaining_visits"`
	LastAccessAt      *time.Time `json:"last_access_at"`
	Status            string     `json:"status"`
}

type ShareVerifyInput struct {
	ViewerID string
	Password string
}

type ShareVerifyResult struct {
	AccessToken string             `json:"access_token"`
	ExpiresAt   time.Time          `json:"expires_at"`
	Share       PublicShareSummary `json:"share"`
}

type PublicLocationResult struct {
	DeviceSN        string     `json:"device_sn"`
	DeviceName      string     `json:"device_name"`
	Battery         int        `json:"battery"`
	GPSState        string     `json:"gps_state"`
	Status          int        `json:"status"`
	LastOnline      *time.Time `json:"last_online"`
	LastFixAt       *time.Time `json:"last_fix_at"`
	Latitude        *float64   `json:"lat,omitempty"`
	Longitude       *float64   `json:"lng,omitempty"`
	Time            *time.Time `json:"time,omitempty"`
	StillSeconds    int        `json:"still_seconds"`
	AccuracyMeters  *float64   `json:"accuracy_m,omitempty"`
	Address         string     `json:"address,omitempty"`
	Activity        string     `json:"activity,omitempty"`
}

type PublicTrackResult struct {
	DeviceSN   string             `json:"device_sn"`
	Tracks     []DeviceTrackPoint `json:"tracks"`
	StartTime  time.Time          `json:"start_time"`
	EndTime    time.Time          `json:"end_time"`
}

type ShareService struct {
	db         *gorm.DB
	deviceRepo *repository.DeviceRepository
	shareRepo  *repository.ShareRepository
}

func NewShareService(db *gorm.DB, deviceRepo *repository.DeviceRepository, shareRepo *repository.ShareRepository) *ShareService {
	return &ShareService{
		db:         db,
		deviceRepo: deviceRepo,
		shareRepo:  shareRepo,
	}
}

func (s *ShareService) ListShares(ctx context.Context, query ShareListQuery) (*ShareListResult, error) {
	page := normalizePage(query.Page)
	pageSize := normalizePageSize(query.PageSize, 50, 200)

	var deviceID *uint64
	if trimmedSN := strings.TrimSpace(query.DeviceSN); trimmedSN != "" {
		device, err := s.deviceRepo.GetByDeviceSN(ctx, trimmedSN)
		if err != nil {
			return nil, err
		}
		if device == nil {
			return &ShareListResult{
				Shares:     []ShareSummary{},
				Pagination: buildPagination(page, pageSize, 0),
			}, nil
		}
		deviceID = &device.ID
	}

	rows, total, err := s.shareRepo.List(ctx, repository.ShareListFilter{
		DeviceID: deviceID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}

	result := make([]ShareSummary, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapShareSummary(row))
	}

	return &ShareListResult{
		Shares:     result,
		Pagination: buildPagination(page, pageSize, total),
	}, nil
}

func (s *ShareService) CreateShare(ctx context.Context, input ShareCreateInput) (*ShareCreateResult, error) {
	if s.db == nil {
		return nil, errors.New("share service requires database")
	}

	device, err := s.deviceRepo.GetByDeviceSN(ctx, strings.TrimSpace(input.DeviceSN))
	if err != nil {
		return nil, err
	}
	if device == nil {
		return nil, ErrDeviceNotFound
	}

	mode := normalizeShareMode(input.ShareMode)
	if mode == "" {
		return nil, ErrInvalidShareMode
	}

	expiresAt := input.ExpiresAt.UTC()
	if !expiresAt.After(time.Now().UTC()) {
		return nil, ErrInvalidShareExpiry
	}

	if input.MaxVisits != nil && *input.MaxVisits <= 0 {
		return nil, errors.New("max_visits must be greater than 0")
	}

	plainPassword := normalizeNullableString(input.Password)
	var passwordHash *string
	if plainPassword != nil {
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(*plainPassword), bcrypt.DefaultCost)
		if hashErr != nil {
			return nil, hashErr
		}
		encoded := string(hash)
		passwordHash = &encoded
	}

	share := model.LocationShare{
		DeviceID:        device.ID,
		CreatedByUserID: input.CreatedByUserID,
		ShareCode:       generateShareCode(),
		ShareMode:       mode,
		PasswordHash:    passwordHash,
		Note:            strings.TrimSpace(input.Note),
		ExpiresAt:       expiresAt,
		MaxVisits:       input.MaxVisits,
	}
	if err := s.shareRepo.Create(ctx, &share); err != nil {
		return nil, err
	}

	summary := mapShareSummary(repository.ShareWithDevice{
		Share:  share,
		Device: *device,
	})

	return &ShareCreateResult{
		Share:    summary,
		Password: plainPassword,
	}, nil
}

func (s *ShareService) RevokeShare(ctx context.Context, shareID uint64) error {
	err := s.shareRepo.Revoke(ctx, shareID, time.Now().UTC())
	if errors.Is(err, repository.ErrLocationShareNotFound) {
		return ErrShareNotFound
	}
	return err
}

func (s *ShareService) GetPublicShare(ctx context.Context, shareCode string) (*PublicShareSummary, error) {
	row, err := s.shareRepo.GetByCode(ctx, shareCode)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrShareNotFound
	}

	summary := mapPublicShareSummary(*row)
	return &summary, nil
}

func (s *ShareService) VerifyPublicShare(ctx context.Context, shareCode string, input ShareVerifyInput) (*ShareVerifyResult, error) {
	row, err := s.shareRepo.GetByCode(ctx, shareCode)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrShareNotFound
	}

	share := row.Share
	now := time.Now().UTC()
	if share.RevokedAt != nil {
		return nil, ErrShareRevoked
	}
	if !share.ExpiresAt.After(now) {
		return nil, ErrShareExpired
	}

	viewerID := strings.TrimSpace(input.ViewerID)
	if viewerID == "" {
		return nil, ErrInvalidViewerID
	}

	if share.PasswordHash != nil {
		if strings.TrimSpace(input.Password) == "" {
			return nil, ErrSharePasswordRequired
		}
		if err := bcrypt.CompareHashAndPassword([]byte(*share.PasswordHash), []byte(input.Password)); err != nil {
			return nil, ErrShareInvalidPassword
		}
	}

	existingSession, err := s.shareRepo.FindReusableSession(ctx, share.ID, viewerID, now)
	if err != nil {
		return nil, err
	}
	if existingSession != nil {
		if touchErr := s.shareRepo.TouchSession(ctx, existingSession.ID, now); touchErr != nil {
			return nil, touchErr
		}

		summary := mapPublicShareSummary(*row)
		return &ShareVerifyResult{
			AccessToken: existingSession.AccessToken,
			ExpiresAt:   existingSession.ExpiresAt,
			Share:       summary,
		}, nil
	}

	accessToken := generateAccessToken()
	sessionExpiresAt := share.ExpiresAt

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var locked model.LocationShare
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", share.ID).
			Take(&locked).Error; err != nil {
			return fmt.Errorf("load location share %d for verify: %w", share.ID, err)
		}

		if locked.RevokedAt != nil {
			return ErrShareRevoked
		}
		if !locked.ExpiresAt.After(now) {
			return ErrShareExpired
		}
		if locked.MaxVisits != nil && locked.VisitCount >= *locked.MaxVisits {
			return ErrShareVisitLimit
		}

		if err := tx.Model(&model.LocationShare{}).
			Where("id = ?", locked.ID).
			Updates(map[string]any{
				"visit_count":     locked.VisitCount + 1,
				"last_access_at":  now,
			}).Error; err != nil {
			return fmt.Errorf("increment share visit count: %w", err)
		}

		session := model.LocationShareSession{
			ShareID:      locked.ID,
			ViewerID:     viewerID,
			AccessToken:  accessToken,
			ExpiresAt:    sessionExpiresAt,
			LastSeenAt:   &now,
			LastAccessAt: &now,
		}
		if err := tx.Create(&session).Error; err != nil {
			return fmt.Errorf("create share session: %w", err)
		}

		share = locked
		share.VisitCount = locked.VisitCount + 1
		share.LastAccessAt = &now
		return nil
	})
	if err != nil {
		return nil, err
	}

	row.Share = share
	summary := mapPublicShareSummary(*row)
	return &ShareVerifyResult{
		AccessToken: accessToken,
		ExpiresAt:   sessionExpiresAt,
		Share:       summary,
	}, nil
}

func (s *ShareService) GetPublicLocation(ctx context.Context, shareCode string, accessToken string) (*PublicLocationResult, error) {
	row, session, err := s.authorizePublicShare(ctx, shareCode, accessToken)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if err := s.shareRepo.TouchSession(ctx, session.ID, now); err != nil {
		return nil, err
	}

	location, err := s.loadPublicLocation(ctx, row.Device)
	if err != nil {
		return nil, err
	}

	return location, nil
}

func (s *ShareService) GetPublicTrack(ctx context.Context, shareCode string, accessToken string) (*PublicTrackResult, error) {
	row, session, err := s.authorizePublicShare(ctx, shareCode, accessToken)
	if err != nil {
		return nil, err
	}

	if row.Share.ShareMode != ShareModeTodayTrack {
		return nil, ErrShareTrackNotAllowed
	}

	now := time.Now()
	startTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).UTC()
	endTime := startTime.Add(24 * time.Hour)

	records, _, err := s.deviceRepo.GetTrackByDeviceID(ctx, row.Device.ID, repository.TrackPageFilter{
		StartTime: &startTime,
		EndTime:   &endTime,
		Page:      1,
		PageSize:  2000,
	})
	if err != nil {
		return nil, err
	}

	if err := s.shareRepo.TouchSession(ctx, session.ID, time.Now().UTC()); err != nil {
		return nil, err
	}

	tracks := make([]DeviceTrackPoint, 0, len(records))
	for _, record := range records {
		tracks = append(tracks, DeviceTrackPoint{
			Latitude:     record.Latitude,
			Longitude:    record.Longitude,
			Time:         record.GPSTime,
			StillSeconds: record.StillSeconds,
		})
	}

	return &PublicTrackResult{
		DeviceSN:  row.Device.DeviceSN,
		Tracks:    tracks,
		StartTime: startTime,
		EndTime:   endTime,
	}, nil
}

func (s *ShareService) ValidatePublicWebsocket(ctx context.Context, shareCode string, accessToken string) (string, error) {
	row, session, err := s.authorizePublicShare(ctx, shareCode, accessToken)
	if err != nil {
		return "", err
	}

	if err := s.shareRepo.TouchSession(ctx, session.ID, time.Now().UTC()); err != nil {
		return "", err
	}

	return row.Device.DeviceSN, nil
}

func (s *ShareService) authorizePublicShare(ctx context.Context, shareCode string, accessToken string) (*repository.ShareWithDevice, *model.LocationShareSession, error) {
	row, err := s.shareRepo.GetByCode(ctx, shareCode)
	if err != nil {
		return nil, nil, err
	}
	if row == nil {
		return nil, nil, ErrShareNotFound
	}

	now := time.Now().UTC()
	if row.Share.RevokedAt != nil {
		return nil, nil, ErrShareRevoked
	}
	if !row.Share.ExpiresAt.After(now) {
		return nil, nil, ErrShareExpired
	}

	session, err := s.shareRepo.GetSessionByToken(ctx, row.Share.ID, accessToken, now)
	if err != nil {
		return nil, nil, err
	}
	if session == nil {
		return nil, nil, ErrShareAccessDenied
	}

	return row, session, nil
}

func (s *ShareService) loadPublicLocation(ctx context.Context, device model.Device) (*PublicLocationResult, error) {
	statusPayload := decodeJSONMap(device.StatusPayload)

	result := &PublicLocationResult{
		DeviceSN:   device.DeviceSN,
		DeviceName: device.Name,
		Battery:    device.Battery,
		GPSState:   device.GPSState,
		Status:     device.Status,
		LastOnline: device.LastOnline,
		LastFixAt:  device.LastFixAt,
		Address:    stringOrEmpty(statusPayload, "address"),
		Activity:   stringOrEmpty(statusPayload, "activity"),
	}
	if accuracy, ok := lookupFloat64(statusPayload, "accuracy_m", "accuracy"); ok {
		result.AccuracyMeters = &accuracy
	}

	if device.LastLatitude != nil && device.LastLongitude != nil && device.LastLocationAt != nil {
		lat := *device.LastLatitude
		lng := *device.LastLongitude
		timeValue := *device.LastLocationAt
		result.Latitude = &lat
		result.Longitude = &lng
		result.Time = &timeValue
		result.StillSeconds = device.LastStillSeconds
		return result, nil
	}

	var record model.GPSRecord
	err := s.db.WithContext(ctx).
		Where("device_id = ?", device.ID).
		Order("gps_time DESC").
		Order("id DESC").
		Take(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, nil
		}

		return nil, fmt.Errorf("load latest gps record for device %s: %w", device.DeviceSN, err)
	}

	lat := record.Latitude
	lng := record.Longitude
	timeValue := record.GPSTime
	result.Latitude = &lat
	result.Longitude = &lng
	result.Time = &timeValue
	result.StillSeconds = record.StillSeconds

	return result, nil
}

func mapShareSummary(row repository.ShareWithDevice) ShareSummary {
	status := shareStatus(row.Share)
	return ShareSummary{
		ID:               row.Share.ID,
		DeviceSN:         row.Device.DeviceSN,
		DeviceName:       row.Device.Name,
		ShareCode:        row.Share.ShareCode,
		ShareMode:        row.Share.ShareMode,
		RequiresPassword: row.Share.PasswordHash != nil,
		Note:             row.Share.Note,
		ExpiresAt:        row.Share.ExpiresAt,
		MaxVisits:        row.Share.MaxVisits,
		VisitCount:       row.Share.VisitCount,
		RemainingVisits:  remainingVisits(row.Share.MaxVisits, row.Share.VisitCount),
		LastAccessAt:     row.Share.LastAccessAt,
		RevokedAt:        row.Share.RevokedAt,
		CreatedAt:        row.Share.CreatedAt,
		Status:           status,
	}
}

func mapPublicShareSummary(row repository.ShareWithDevice) PublicShareSummary {
	return PublicShareSummary{
		ShareCode:        row.Share.ShareCode,
		DeviceSN:         row.Device.DeviceSN,
		DeviceName:       row.Device.Name,
		ShareMode:        row.Share.ShareMode,
		RequiresPassword: row.Share.PasswordHash != nil,
		ExpiresAt:        row.Share.ExpiresAt,
		MaxVisits:        row.Share.MaxVisits,
		VisitCount:       row.Share.VisitCount,
		RemainingVisits:  remainingVisits(row.Share.MaxVisits, row.Share.VisitCount),
		LastAccessAt:     row.Share.LastAccessAt,
		Status:           shareStatus(row.Share),
	}
}

func remainingVisits(maxVisits *int, visitCount int) *int {
	if maxVisits == nil {
		return nil
	}

	remaining := *maxVisits - visitCount
	if remaining < 0 {
		remaining = 0
	}

	return &remaining
}

func shareStatus(share model.LocationShare) string {
	now := time.Now().UTC()
	if share.RevokedAt != nil {
		return "revoked"
	}
	if !share.ExpiresAt.After(now) {
		return "expired"
	}
	if share.MaxVisits != nil && share.VisitCount >= *share.MaxVisits {
		return "quota_used"
	}
	if share.ExpiresAt.Sub(now) <= 30*time.Minute {
		return "expiring"
	}
	return "active"
}

func normalizeShareMode(value string) string {
	switch strings.TrimSpace(value) {
	case ShareModeLiveOnly:
		return ShareModeLiveOnly
	case ShareModeTodayTrack:
		return ShareModeTodayTrack
	default:
		return ""
	}
}

func generateShareCode() string {
	return randomHex(5)
}

func generateAccessToken() string {
	return randomHex(24)
}

func randomHex(byteLen int) string {
	buffer := make([]byte, byteLen)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	}

	return hex.EncodeToString(buffer)
}

func decodeJSONMap(raw datatypes.JSON) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil || payload == nil {
		return map[string]any{}
	}

	return payload
}

func stringOrEmpty(payload map[string]any, keys ...string) string {
	value, ok := lookupString(payload, keys...)
	if !ok {
		return ""
	}

	return value
}
