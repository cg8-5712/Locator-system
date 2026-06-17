package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	mqttclient "locator/internal/mqtt"
	"locator/internal/service"
)

type mqttService interface {
	Enabled() bool
	Connected() bool
	Topics() []string
	Publish(ctx context.Context, topic string, payload []byte, qos byte, retained bool) error
	RecentMessages(limit int) []mqttclient.MessageSnapshot
}

type databaseService interface {
	Driver() string
	PingContext(ctx context.Context) error
}

type deviceQueryService interface {
	ListDevices(ctx context.Context, limit int) ([]service.DeviceSummary, error)
	GetDevice(ctx context.Context, id uint64) (*service.DeviceSummary, error)
	GetTrack(ctx context.Context, deviceID uint64, startTime *time.Time, endTime *time.Time, limit int) ([]service.DeviceTrackPoint, error)
}

type mqttPublishRequest struct {
	Topic    string          `json:"topic" binding:"required"`
	QoS      uint8           `json:"qos"`
	Retained bool            `json:"retained"`
	Payload  json.RawMessage `json:"payload" binding:"required"`
}

func NewRouter(appLogger *slog.Logger, mqttSvc mqttService, dbSvc databaseService, deviceSvc deviceQueryService) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestLogger(appLogger))

	router.GET("/health", func(c *gin.Context) {
		dbConnected := false
		dbDriver := ""
		if dbSvc != nil {
			dbDriver = dbSvc.Driver()

			pingCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()

			dbConnected = dbSvc.PingContext(pingCtx) == nil
		}

		ok(c, gin.H{
			"status":             "ok",
			"time":               time.Now().UTC(),
			"database_driver":    dbDriver,
			"database_connected": dbConnected,
			"mqtt_enabled":       mqttSvc.Enabled(),
			"mqtt_connected":     mqttSvc.Connected(),
		})
	})

	apiGroup := router.Group("/api")
	{
		apiGroup.GET("/devices", func(c *gin.Context) {
			if deviceSvc == nil {
				fail(c, http.StatusServiceUnavailable, "device service is unavailable")
				return
			}

			limit := parsePositiveInt(c.Query("limit"), 50, 200)
			devices, err := deviceSvc.ListDevices(c.Request.Context(), limit)
			if err != nil {
				fail(c, http.StatusInternalServerError, err.Error())
				return
			}

			ok(c, gin.H{
				"devices": devices,
			})
		})

		apiGroup.GET("/devices/:id", func(c *gin.Context) {
			if deviceSvc == nil {
				fail(c, http.StatusServiceUnavailable, "device service is unavailable")
				return
			}

			deviceID, valid := parseUint64Param(c, "id")
			if !valid {
				return
			}

			device, err := deviceSvc.GetDevice(c.Request.Context(), deviceID)
			if err != nil {
				fail(c, http.StatusInternalServerError, err.Error())
				return
			}

			if device == nil {
				fail(c, http.StatusNotFound, "device not found")
				return
			}

			ok(c, device)
		})

		apiGroup.GET("/devices/:id/tracks", func(c *gin.Context) {
			if deviceSvc == nil {
				fail(c, http.StatusServiceUnavailable, "device service is unavailable")
				return
			}

			deviceID, valid := parseUint64Param(c, "id")
			if !valid {
				return
			}

			startTime, err := parseOptionalTime(c.Query("start_time"))
			if err != nil {
				fail(c, http.StatusBadRequest, "invalid start_time: "+err.Error())
				return
			}

			endTime, err := parseOptionalTime(c.Query("end_time"))
			if err != nil {
				fail(c, http.StatusBadRequest, "invalid end_time: "+err.Error())
				return
			}

			limit := parsePositiveInt(c.Query("limit"), 500, 5000)
			track, err := deviceSvc.GetTrack(c.Request.Context(), deviceID, startTime, endTime, limit)
			if err != nil {
				if errors.Is(err, service.ErrDeviceNotFound) {
					fail(c, http.StatusNotFound, err.Error())
					return
				}

				fail(c, http.StatusBadRequest, err.Error())
				return
			}

			ok(c, gin.H{
				"tracks": track,
			})
		})

		apiGroup.GET("/mqtt/status", func(c *gin.Context) {
			ok(c, gin.H{
				"enabled":   mqttSvc.Enabled(),
				"connected": mqttSvc.Connected(),
				"topics":    mqttSvc.Topics(),
			})
		})

		apiGroup.GET("/mqtt/messages", func(c *gin.Context) {
			limit := parsePositiveInt(c.Query("limit"), 20, 100)
			ok(c, gin.H{
				"messages": mqttSvc.RecentMessages(limit),
			})
		})

		apiGroup.POST("/debug/mqtt/publish", func(c *gin.Context) {
			var req mqttPublishRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				fail(c, http.StatusBadRequest, "invalid publish request: "+err.Error())
				return
			}

			if !mqttSvc.Enabled() {
				fail(c, http.StatusServiceUnavailable, "mqtt is disabled")
				return
			}

			if err := mqttSvc.Publish(c.Request.Context(), req.Topic, req.Payload, byte(req.QoS), req.Retained); err != nil {
				if errors.Is(err, mqttclient.ErrNotConnected) {
					fail(c, http.StatusServiceUnavailable, err.Error())
					return
				}

				fail(c, http.StatusBadGateway, err.Error())
				return
			}

			ok(c, gin.H{
				"topic":     req.Topic,
				"qos":       req.QoS,
				"retained":  req.Retained,
				"published": true,
			})
		})
	}

	return router
}

func requestLogger(appLogger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		appLogger.Info("http request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency", time.Since(start),
			"client_ip", c.ClientIP(),
		)
	}
}

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

func fail(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, gin.H{
		"success": false,
		"error":   message,
	})
}

func parsePositiveInt(raw string, defaultValue int, maxValue int) int {
	if raw == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return defaultValue
	}

	if value > maxValue {
		return maxValue
	}

	return value
}

func parseUint64Param(c *gin.Context, key string) (uint64, bool) {
	raw := c.Param(key)
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 {
		fail(c, http.StatusBadRequest, "invalid "+key)
		return 0, false
	}

	return value, true
}

func parseOptionalTime(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}

	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			parsed = parsed.UTC()
			return &parsed, nil
		}
	}

	return nil, errors.New("expected RFC3339 or 2006-01-02 15:04:05")
}
