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
	ID         uint64     `json:"id" gorm:"primaryKey"`
	DeviceSN   string     `json:"device_sn" gorm:"column:device_sn;size:64;uniqueIndex;not null"`
	IMEI       string     `json:"imei" gorm:"column:imei;size:32;uniqueIndex"`
	ICCID      string     `json:"iccid" gorm:"column:iccid;size:32;index"`
	Name       string     `json:"name" gorm:"size:64"`
	Status     int        `json:"status" gorm:"not null;default:0"`
	Battery    int        `json:"battery" gorm:"not null;default:0"`
	LastOnline *time.Time `json:"last_online" gorm:"column:last_online"`
	CreatedAt  time.Time  `json:"created_at" gorm:"column:created_at;not null;autoCreateTime"`
}

func (Device) TableName() string {
	return "devices"
}

type GPSRecord struct {
	ID        uint64    `json:"id" gorm:"primaryKey"`
	DeviceID  uint64    `json:"device_id" gorm:"column:device_id;not null;index:idx_gps_records_device_time,priority:1"`
	Latitude  float64   `json:"latitude" gorm:"column:latitude;not null"`
	Longitude float64   `json:"longitude" gorm:"column:longitude;not null"`
	GPSTime   time.Time `json:"gps_time" gorm:"column:gps_time;not null;index:idx_gps_records_device_time,priority:2"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;not null;autoCreateTime"`
}

func (GPSRecord) TableName() string {
	return "gps_records"
}

type Fence struct {
	ID        uint64         `json:"id" gorm:"primaryKey"`
	DeviceID  uint64         `json:"device_id" gorm:"column:device_id;not null;index"`
	Name      string         `json:"name" gorm:"size:64;not null"`
	Polygon   datatypes.JSON `json:"polygon" gorm:"column:polygon;not null"`
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at;not null;autoCreateTime"`
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

func AutoMigrateModels() []any {
	return []any{
		&User{},
		&Device{},
		&GPSRecord{},
		&Fence{},
		&Alarm{},
	}
}
