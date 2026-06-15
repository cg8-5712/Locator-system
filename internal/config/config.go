package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv          string
	LogLevel        string
	ShutdownTimeout time.Duration
	HTTP            HTTPConfig
	MQTT            MQTTConfig
}

type HTTPConfig struct {
	Addr string
}

type MQTTConfig struct {
	Enabled          bool
	Broker           string
	ClientID         string
	Username         string
	Password         string
	Topics           []string
	QoS              byte
	ConnectTimeout   time.Duration
	OperationTimeout time.Duration
}

func Load() Config {
	return Config{
		AppEnv:          getEnv("APP_ENV", "development"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 10*time.Second),
		HTTP: HTTPConfig{
			Addr: getEnv("HTTP_ADDR", ":8080"),
		},
		MQTT: MQTTConfig{
			Enabled:          getBoolEnv("MQTT_ENABLED", false),
			Broker:           getEnv("MQTT_BROKER", "tcp://127.0.0.1:1883"),
			ClientID:         getEnv("MQTT_CLIENT_ID", "locator-backend"),
			Username:         getEnv("MQTT_USERNAME", ""),
			Password:         getEnv("MQTT_PASSWORD", ""),
			Topics:           getCSVEnv("MQTT_TOPICS", []string{"device/+/gps", "device/+/status", "device/+/alarm"}),
			QoS:              getByteEnv("MQTT_QOS", 1),
			ConnectTimeout:   getDurationEnv("MQTT_CONNECT_TIMEOUT", 10*time.Second),
			OperationTimeout: getDurationEnv("MQTT_OPERATION_TIMEOUT", 5*time.Second),
		},
	}
}

func getEnv(key string, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return defaultValue
	}

	return value
}

func getBoolEnv(key string, defaultValue bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}

	return parsed
}

func getByteEnv(key string, defaultValue byte) byte {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseUint(value, 10, 8)
	if err != nil {
		return defaultValue
	}

	return byte(parsed)
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return defaultValue
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}

	return parsed
}

func getCSVEnv(key string, defaultValue []string) []string {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return append([]string(nil), defaultValue...)
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		result = append(result, trimmed)
	}

	if len(result) == 0 {
		return append([]string(nil), defaultValue...)
	}

	return result
}
