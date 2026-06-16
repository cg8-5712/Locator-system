package database

import (
	"context"
	"path/filepath"
	"testing"

	"locator/internal/config"
	"locator/internal/model"
	"locator/pkg/logger"
)

func TestOpenSQLiteAutoMigrate(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "data", "locator.db")

	store, err := Open(config.DatabaseConfig{
		Driver:          "sqlite",
		DSN:             dbPath,
		AutoMigrate:     true,
		MaxIdleConns:    1,
		MaxOpenConns:    1,
		ConnMaxIdleTime: 0,
		ConnMaxLifetime: 0,
	}, logger.New("error"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	}()

	if err := store.PingContext(context.Background()); err != nil {
		t.Fatalf("PingContext() error = %v", err)
	}

	models := []any{
		&model.User{},
		&model.Device{},
		&model.GPSRecord{},
		&model.Fence{},
		&model.Alarm{},
	}

	for _, table := range models {
		if !store.DB().Migrator().HasTable(table) {
			t.Fatalf("expected table for %T to exist", table)
		}
	}
}
