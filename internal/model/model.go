package model

import (
	"time"

	"gorm.io/datatypes"
)

type User struct {
	ID           uint64    `json:"id" gorm:"primaryKey"`
	Username     string    `json:"username" gorm:"size:64;uniqueIndex;not null"`
	PasswordHash string    `json:"password_hash" gorm:"column:password_hash;type:text;not null"`
	Role         string    `json:"role" gorm:"size:16;not null;default:user"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at;not null;autoCreateTime"`
}

func (User) TableName() string {
	return "users"
}

type Device struct {
	ID              uint64         `json:"id" gorm:"primaryKey"`
	DeviceSN        string         `json:"device_sn" gorm:"column:device_sn;size:64;uniqueIndex;not null"`
	IMEI            *string        `json:"imei" gorm:"column:imei;size:32;uniqueIndex"`
	ICCID           *string        `json:"iccid" gorm:"column:iccid;size:32;index"`
	Name            string         `json:"name" gorm:"size:64"`
	TopicPrefix     string         `json:"topic_prefix" gorm:"column:topic_prefix;size:32;not null;default:locator"`
	GPSState        string         `json:"gps_state" gorm:"column:gps_state;size:32"`
	Status          int            `json:"status" gorm:"not null;default:0"`
	Battery         int            `json:"battery" gorm:"not null;default:0"`
	StatusPayload   datatypes.JSON `json:"status_payload" gorm:"column:status_payload"`
	ConfigPayload   datatypes.JSON `json:"config_payload" gorm:"column:config_payload"`
	StatusUpdatedAt *time.Time     `json:"status_updated_at" gorm:"column:status_updated_at"`
	ConfigUpdatedAt *time.Time     `json:"config_updated_at" gorm:"column:config_updated_at"`
	LastLatitude    *float64       `json:"last_latitude" gorm:"column:last_latitude"`
	LastLongitude   *float64       `json:"last_longitude" gorm:"column:last_longitude"`
	LastLocationAt  *time.Time     `json:"last_location_at" gorm:"column:last_location_at"`
	LastStillSeconds int           `json:"last_still_seconds" gorm:"column:last_still_seconds;not null;default:0"`
	LastFixAt       *time.Time     `json:"last_fix_at" gorm:"column:last_fix_at"`
	LastOnline      *time.Time     `json:"last_online" gorm:"column:last_online"`
	CreatedAt       time.Time      `json:"created_at" gorm:"column:created_at;not null;autoCreateTime"`
}

func (Device) TableName() string {
	return "devices"
}

type GPSRecord struct {
	ID           uint64    `json:"id" gorm:"primaryKey"`
	DeviceID     uint64    `json:"device_id" gorm:"column:device_id;not null;index:idx_gps_records_device_time,priority:1"`
	Latitude     float64   `json:"latitude" gorm:"column:latitude;not null"`
	Longitude    float64   `json:"longitude" gorm:"column:longitude;not null"`
	GPSTime      time.Time `json:"gps_time" gorm:"column:gps_time;not null;index:idx_gps_records_device_time,priority:2"`
	StillSeconds int       `json:"still_seconds" gorm:"column:still_seconds;not null;default:0"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at;not null;autoCreateTime"`
}

func (GPSRecord) TableName() string {
	return "gps_records"
}

type Fence struct {
	ID            uint64         `json:"id" gorm:"primaryKey"`
	DeviceID      uint64         `json:"device_id" gorm:"column:device_id;not null;index"`
	Name          string         `json:"name" gorm:"size:64;not null"`
	Polygon       datatypes.JSON `json:"polygon" gorm:"column:polygon;not null"`
	LastInside    *bool          `json:"last_inside" gorm:"column:last_inside"`
	LastCheckedAt *time.Time     `json:"last_checked_at" gorm:"column:last_checked_at"`
	CreatedAt     time.Time      `json:"created_at" gorm:"column:created_at;not null;autoCreateTime"`
}

func (Fence) TableName() string {
	return "fences"
}

type Alarm struct {
	ID        uint64    `json:"id" gorm:"primaryKey"`
	DeviceID  uint64    `json:"device_id" gorm:"column:device_id;not null;index"`
	Type      string    `json:"type" gorm:"column:type;size:32;not null;index"`
	Content   string    `json:"content" gorm:"column:content;type:text;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;not null;autoCreateTime"`
}

func (Alarm) TableName() string {
	return "alarms"
}

type LocationShare struct {
	ID              uint64     `json:"id" gorm:"primaryKey"`
	DeviceID        uint64     `json:"device_id" gorm:"column:device_id;not null;index"`
	CreatedByUserID *uint64    `json:"created_by_user_id" gorm:"column:created_by_user_id;index"`
	ShareCode       string     `json:"share_code" gorm:"column:share_code;size:32;uniqueIndex;not null"`
	ShareMode       string     `json:"share_mode" gorm:"column:share_mode;size:32;not null"`
	PasswordHash    *string    `json:"password_hash" gorm:"column:password_hash;type:text"`
	Note            string     `json:"note" gorm:"column:note;type:text"`
	ExpiresAt       time.Time  `json:"expires_at" gorm:"column:expires_at;not null;index"`
	MaxVisits       *int       `json:"max_visits" gorm:"column:max_visits"`
	VisitCount      int        `json:"visit_count" gorm:"column:visit_count;not null;default:0"`
	LastAccessAt    *time.Time `json:"last_access_at" gorm:"column:last_access_at"`
	RevokedAt       *time.Time `json:"revoked_at" gorm:"column:revoked_at;index"`
	CreatedAt       time.Time  `json:"created_at" gorm:"column:created_at;not null;autoCreateTime"`
}

func (LocationShare) TableName() string {
	return "location_shares"
}

type LocationShareSession struct {
	ID          uint64     `json:"id" gorm:"primaryKey"`
	ShareID      uint64     `json:"share_id" gorm:"column:share_id;not null;index"`
	ViewerID     string     `json:"viewer_id" gorm:"column:viewer_id;size:128;not null;index"`
	AccessToken  string     `json:"access_token" gorm:"column:access_token;size:128;uniqueIndex;not null"`
	ExpiresAt    time.Time  `json:"expires_at" gorm:"column:expires_at;not null;index"`
	LastSeenAt   *time.Time `json:"last_seen_at" gorm:"column:last_seen_at"`
	LastAccessAt *time.Time `json:"last_access_at" gorm:"column:last_access_at"`
	CreatedAt    time.Time  `json:"created_at" gorm:"column:created_at;not null;autoCreateTime"`
}

func (LocationShareSession) TableName() string {
	return "location_share_sessions"
}

func AutoMigrateModels() []any {
	return []any{
		&User{},
		&Device{},
		&GPSRecord{},
		&Fence{},
		&Alarm{},
		&LocationShare{},
		&LocationShareSession{},
	}
}
