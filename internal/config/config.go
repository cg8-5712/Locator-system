package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv          string
	LogLevel        string
	ShutdownTimeout time.Duration
	HTTP            HTTPConfig
	Database        DatabaseConfig
	Auth            AuthConfig
	Alarm           AlarmConfig
	Offline         OfflineConfig
	MQTT            MQTTConfig
}

type HTTPConfig struct {
	Addr string
}

type DatabaseConfig struct {
	Driver          string
	DSN             string
	AutoMigrate     bool
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
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

type AuthConfig struct {
	Enabled                bool
	JWTSecret              string
	TokenTTL               time.Duration
	BootstrapAdminUsername string
	BootstrapAdminPassword string
}

type OfflineConfig struct {
	CheckInterval time.Duration
	OfflineAfter  time.Duration
}

type AlarmConfig struct {
	DedupeWindow time.Duration
}

func Load() Config {
	loadEnvFiles(".env", ".env.local")

	dbDriver := normalizeDatabaseDriver(getEnv("DB_DRIVER", "sqlite"))

	return Config{
		AppEnv:          getEnv("APP_ENV", "development"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 10*time.Second),
		HTTP: HTTPConfig{
			Addr: getEnv("HTTP_ADDR", ":8080"),
		},
		Database: DatabaseConfig{
			Driver:          dbDriver,
			DSN:             getEnv("DB_DSN", defaultDatabaseDSN(dbDriver)),
			AutoMigrate:     getBoolEnv("DB_AUTO_MIGRATE", true),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 20),
			ConnMaxIdleTime: getDurationEnv("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		},
		Auth: AuthConfig{
			Enabled:                getBoolEnv("AUTH_ENABLED", true),
			JWTSecret:              getEnv("AUTH_JWT_SECRET", "change-me"),
			TokenTTL:               getDurationEnv("AUTH_TOKEN_TTL", 24*time.Hour),
			BootstrapAdminUsername: getEnv("AUTH_BOOTSTRAP_ADMIN_USERNAME", "admin"),
			BootstrapAdminPassword: getEnv("AUTH_BOOTSTRAP_ADMIN_PASSWORD", "admin123456"),
		},
		Alarm: AlarmConfig{
			DedupeWindow: getDurationEnv("ALARM_DEDUPE_WINDOW", 5*time.Minute),
		},
		Offline: OfflineConfig{
			CheckInterval: getDurationEnv("OFFLINE_CHECK_INTERVAL", time.Minute),
			OfflineAfter:  getDurationEnv("OFFLINE_AFTER", 5*time.Minute),
		},
		MQTT: MQTTConfig{
			Enabled:  getBoolEnv("MQTT_ENABLED", false),
			Broker:   getEnv("MQTT_BROKER", "tcp://127.0.0.1:1883"),
			ClientID: getEnv("MQTT_CLIENT_ID", "locator-backend"),
			Username: getEnv("MQTT_USERNAME", ""),
			Password: getEnv("MQTT_PASSWORD", ""),
			Topics: getCSVEnv("MQTT_TOPICS", []string{
				"device/+/gps",
				"device/+/status",
				"device/+/alarm",
				"locator/+/location",
				"locator/+/status",
				"locator/+/config",
				"locator/+/test",
			}),
			QoS:              getByteEnv("MQTT_QOS", 1),
			ConnectTimeout:   getDurationEnv("MQTT_CONNECT_TIMEOUT", 10*time.Second),
			OperationTimeout: getDurationEnv("MQTT_OPERATION_TIMEOUT", 5*time.Second),
		},
	}
}

func loadEnvFiles(paths ...string) {
	for _, path := range paths {
		if err := loadEnvFile(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			fmt.Fprintf(os.Stderr, "config: skip env file %s: %v\n", path, err)
		}
	}
}

func loadEnvFile(path string) error {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++

		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\uFEFF"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("invalid line %d", lineNumber)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("empty key on line %d", lineNumber)
		}

		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s from line %d: %w", key, lineNumber, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan env file: %w", err)
	}

	return nil
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

func getIntEnv(key string, defaultValue int) int {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return defaultValue
	}

	return parsed
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

func normalizeDatabaseDriver(driver string) string {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "", "sqlite", "sqlite3":
		return "sqlite"
	case "postgres", "postgresql", "psql":
		return "postgres"
	default:
		return strings.ToLower(strings.TrimSpace(driver))
	}
}

func defaultDatabaseDSN(driver string) string {
	if driver == "postgres" {
		return "host=127.0.0.1 user=postgres password=postgres dbname=locator port=5432 sslmode=disable TimeZone=UTC"
	}

	return "locator.db"
}
