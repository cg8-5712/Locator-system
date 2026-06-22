package service

import "time"

type RealtimePublisher interface {
	PublishLocation(event LocationEvent)
	PublishDeviceStatus(event DeviceStatusEvent)
	PublishAlarm(event AlarmEvent)
}

type LocationEvent struct {
	DeviceSN     string    `json:"device_sn"`
	TopicPrefix  string    `json:"topic_prefix"`
	Latitude     float64   `json:"lat"`
	Longitude    float64   `json:"lng"`
	Time         time.Time `json:"time"`
	StillSeconds int       `json:"still_seconds"`
	GPSState     string    `json:"gps_state"`
	Status       int       `json:"status"`
}

type DeviceStatusEvent struct {
	DeviceSN    string     `json:"device_sn"`
	TopicPrefix string     `json:"topic_prefix"`
	Status      int        `json:"status"`
	GPSState    string     `json:"gps_state"`
	Battery     int        `json:"battery"`
	IMEI        string     `json:"imei,omitempty"`
	ICCID       string     `json:"iccid,omitempty"`
	LastOnline  *time.Time `json:"last_online,omitempty"`
	LastFixAt   *time.Time `json:"last_fix_at,omitempty"`
}

type AlarmEvent struct {
	DeviceSN  string    `json:"device_sn"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
