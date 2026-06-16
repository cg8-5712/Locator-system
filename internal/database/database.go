package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	sqlite "github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"locator/internal/config"
	"locator/internal/model"
)

type Store struct {
	db     *gorm.DB
	sqlDB  *sql.DB
	driver string
	logger *slog.Logger
}

func Open(cfg config.DatabaseConfig, appLogger *slog.Logger) (*Store, error) {
	if appLogger == nil {
		appLogger = slog.Default()
	}

	driver, dialector, err := newDialector(cfg)
	if err != nil {
		return nil, err
	}

	gormDB, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", driver, err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("get %s sql db: %w", driver, err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	if cfg.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	store := &Store{
		db:     gormDB,
		sqlDB:  sqlDB,
		driver: driver,
		logger: appLogger,
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := store.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping %s database: %w", driver, err)
	}

	if cfg.AutoMigrate {
		if err := gormDB.AutoMigrate(model.AutoMigrateModels()...); err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("auto migrate %s database: %w", driver, err)
		}
	}

	appLogger.Info("database connected",
		"driver", driver,
		"auto_migrate", cfg.AutoMigrate,
	)

	return store, nil
}

func (s *Store) DB() *gorm.DB {
	return s.db
}

func (s *Store) Driver() string {
	return s.driver
}

func (s *Store) PingContext(ctx context.Context) error {
	return s.sqlDB.PingContext(ctx)
}

func (s *Store) Close() error {
	if err := s.sqlDB.Close(); err != nil {
		return fmt.Errorf("close %s database: %w", s.driver, err)
	}

	s.logger.Info("database disconnected", "driver", s.driver)
	return nil
}

func newDialector(cfg config.DatabaseConfig) (string, gorm.Dialector, error) {
	driver := normalizeDriver(cfg.Driver)
	dsn := strings.TrimSpace(cfg.DSN)

	if dsn == "" {
		return "", nil, fmt.Errorf("database dsn is required for driver %s", driver)
	}

	switch driver {
	case "sqlite":
		if err := ensureSQLitePath(dsn); err != nil {
			return "", nil, fmt.Errorf("prepare sqlite database path: %w", err)
		}

		return driver, sqlite.Open(dsn), nil
	case "postgres":
		return driver, postgres.Open(dsn), nil
	default:
		return "", nil, fmt.Errorf("unsupported database driver %q", cfg.Driver)
	}
}

func normalizeDriver(driver string) string {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "", "sqlite", "sqlite3":
		return "sqlite"
	case "postgres", "postgresql", "psql":
		return "postgres"
	default:
		return strings.ToLower(strings.TrimSpace(driver))
	}
}

func ensureSQLitePath(dsn string) error {
	path := strings.TrimSpace(dsn)
	if path == "" {
		return nil
	}

	if strings.HasPrefix(path, "file:") {
		path = strings.TrimPrefix(path, "file:")
	}

	if index := strings.Index(path, "?"); index >= 0 {
		path = path[:index]
	}

	if path == "" || path == ":memory:" || strings.Contains(strings.ToLower(dsn), "mode=memory") {
		return nil
	}

	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	return nil
}
