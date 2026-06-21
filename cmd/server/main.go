package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"locator/internal/api"
	"locator/internal/config"
	"locator/internal/database"
	mqttclient "locator/internal/mqtt"
	"locator/internal/repository"
	"locator/internal/service"
	"locator/pkg/logger"
)

func main() {
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	appLogger := logger.New(cfg.LogLevel)

	if err := run(rootCtx, cfg, appLogger); err != nil {
		appLogger.Error("application stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(rootCtx context.Context, cfg config.Config, appLogger *slog.Logger) error {
	dbStore, err := database.Open(cfg.Database, appLogger)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() {
		if closeErr := dbStore.Close(); closeErr != nil {
			appLogger.Error("close database", "error", closeErr)
		}
	}()

	deviceRepo := repository.NewDeviceRepository(dbStore.DB())
	deviceSvc := service.NewDeviceService(deviceRepo)
	alarmRepo := repository.NewAlarmRepository(dbStore.DB())
	alarmSvc := service.NewAlarmService(alarmRepo)

	mqttProcessor := service.NewMQTTMessageProcessor(dbStore.DB(), appLogger)
	mqttSvc := mqttclient.New(cfg.MQTT, appLogger, mqttProcessor)
	if err := mqttSvc.Start(rootCtx); err != nil {
		return fmt.Errorf("start mqtt service: %w", err)
	}
	defer mqttSvc.Close()

	router := api.NewRouter(appLogger, mqttSvc, dbStore, deviceSvc, alarmSvc)
	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrCh := make(chan error, 1)
	go func() {
		appLogger.Info("http server listening", "addr", cfg.HTTP.Addr)
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
			return
		}
		serverErrCh <- nil
	}()

	select {
	case <-rootCtx.Done():
		appLogger.Info("shutdown signal received")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}

		return nil
	case err := <-serverErrCh:
		if err != nil {
			return fmt.Errorf("run http server: %w", err)
		}

		return nil
	}
}
