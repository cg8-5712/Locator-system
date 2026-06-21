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

type deviceService interface {
	ListDevices(ctx context.Context, query service.DeviceListQuery) (*service.DeviceListResult, error)
	GetDevice(ctx context.Context, deviceSN string) (*service.DeviceSummary, error)
	GetTrack(ctx context.Context, deviceSN string, query service.TrackQuery) (*service.DeviceTrackResult, error)
	CreateDevice(ctx context.Context, input service.DeviceCreateInput) (*service.DeviceSummary, error)
	UpdateDevice(ctx context.Context, deviceSN string, input service.DeviceUpdateInput) (*service.DeviceSummary, error)
	DeleteDevice(ctx context.Context, deviceSN string) error
}

type alarmService interface {
	ListAlarms(ctx context.Context, query service.AlarmListQuery) (*service.AlarmListResult, error)
}

type mqttPublishRequest struct {
	Topic    string          `json:"topic" binding:"required"`
	QoS      uint8           `json:"qos"`
	Retained bool            `json:"retained"`
	Payload  json.RawMessage `json:"payload" binding:"required"`
}

type deviceUpsertRequest struct {
	IMEI    *string `json:"imei"`
	ICCID   *string `json:"iccid"`
	Name    *string `json:"name"`
	Status  *int    `json:"status"`
	Battery *int    `json:"battery"`
}

type deviceCreateRequest struct {
	DeviceSN string `json:"device_sn" binding:"required"`
	deviceUpsertRequest
}

func NewRouter(appLogger *slog.Logger, mqttSvc mqttService, dbSvc databaseService, deviceSvc deviceService, alarmSvc alarmService) *gin.Engine {
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

			status, err := parseOptionalInt(c.Query("status"))
			if err != nil {
				fail(c, http.StatusBadRequest, "invalid status: "+err.Error())
				return
			}

			result, err := deviceSvc.ListDevices(c.Request.Context(), service.DeviceListQuery{
				DeviceSN: c.Query("device_sn"),
				IMEI:     c.Query("imei"),
				ICCID:    c.Query("iccid"),
				Name:     c.Query("name"),
				Status:   status,
				Page:     parsePositiveInt(c.Query("page"), 1, 100000),
				PageSize: parsePositiveInt(c.Query("page_size"), 20, 200),
			})
			if err != nil {
				fail(c, http.StatusInternalServerError, err.Error())
				return
			}

			ok(c, result)
		})

		apiGroup.POST("/devices", func(c *gin.Context) {
			if deviceSvc == nil {
				fail(c, http.StatusServiceUnavailable, "device service is unavailable")
				return
			}

			var req deviceCreateRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				fail(c, http.StatusBadRequest, "invalid create device request: "+err.Error())
				return
			}

			device, err := deviceSvc.CreateDevice(c.Request.Context(), service.DeviceCreateInput{
				DeviceSN: req.DeviceSN,
				IMEI:     req.IMEI,
				ICCID:    req.ICCID,
				Name:     req.Name,
				Status:   req.Status,
				Battery:  req.Battery,
			})
			if err != nil {
				handleDeviceServiceError(c, err)
				return
			}

			c.JSON(http.StatusCreated, gin.H{
				"success": true,
				"data":    device,
			})
		})

		apiGroup.GET("/devices/:device_sn", func(c *gin.Context) {
			if deviceSvc == nil {
				fail(c, http.StatusServiceUnavailable, "device service is unavailable")
				return
			}

			device, err := deviceSvc.GetDevice(c.Request.Context(), c.Param("device_sn"))
			if err != nil {
				handleDeviceServiceError(c, err)
				return
			}

			ok(c, device)
		})

		apiGroup.PUT("/devices/:device_sn", func(c *gin.Context) {
			if deviceSvc == nil {
				fail(c, http.StatusServiceUnavailable, "device service is unavailable")
				return
			}

			var req deviceUpsertRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				fail(c, http.StatusBadRequest, "invalid update device request: "+err.Error())
				return
			}

			device, err := deviceSvc.UpdateDevice(c.Request.Context(), c.Param("device_sn"), service.DeviceUpdateInput{
				IMEI:    req.IMEI,
				ICCID:   req.ICCID,
				Name:    req.Name,
				Status:  req.Status,
				Battery: req.Battery,
			})
			if err != nil {
				handleDeviceServiceError(c, err)
				return
			}

			ok(c, device)
		})

		apiGroup.GET("/devices/:device_sn/tracks", func(c *gin.Context) {
			if deviceSvc == nil {
				fail(c, http.StatusServiceUnavailable, "device service is unavailable")
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

			result, err := deviceSvc.GetTrack(c.Request.Context(), c.Param("device_sn"), service.TrackQuery{
				StartTime: startTime,
				EndTime:   endTime,
				Page:      parsePositiveInt(c.Query("page"), 1, 100000),
				PageSize:  parsePositiveInt(c.Query("page_size"), 100, 500),
			})
			if err != nil {
				handleDeviceServiceError(c, err)
				return
			}

			ok(c, result)
		})

		apiGroup.DELETE("/devices/:device_sn", func(c *gin.Context) {
			if deviceSvc == nil {
				fail(c, http.StatusServiceUnavailable, "device service is unavailable")
				return
			}

			deviceSN := c.Param("device_sn")
			if err := deviceSvc.DeleteDevice(c.Request.Context(), deviceSN); err != nil {
				handleDeviceServiceError(c, err)
				return
			}

			ok(c, gin.H{
				"deleted":   true,
				"device_sn": deviceSN,
			})
		})

		apiGroup.GET("/alarms", func(c *gin.Context) {
			if alarmSvc == nil {
				fail(c, http.StatusServiceUnavailable, "alarm service is unavailable")
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

			result, err := alarmSvc.ListAlarms(c.Request.Context(), service.AlarmListQuery{
				DeviceSN:  c.Query("device_sn"),
				Type:      c.Query("type"),
				StartTime: startTime,
				EndTime:   endTime,
				Page:      parsePositiveInt(c.Query("page"), 1, 100000),
				PageSize:  parsePositiveInt(c.Query("page_size"), 20, 200),
			})
			if err != nil {
				handleAlarmServiceError(c, err)
				return
			}

			ok(c, result)
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

func handleDeviceServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrDeviceNotFound):
		fail(c, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrDeviceSNConflict), errors.Is(err, service.ErrIMEIConflict):
		fail(c, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrNoDeviceFieldChange), errors.Is(err, service.ErrInvalidTimeRange):
		fail(c, http.StatusBadRequest, err.Error())
	default:
		fail(c, http.StatusBadRequest, err.Error())
	}
}

func handleAlarmServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidTimeRange):
		fail(c, http.StatusBadRequest, err.Error())
	default:
		fail(c, http.StatusInternalServerError, err.Error())
	}
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

func parseOptionalInt(raw string) (*int, error) {
	if raw == "" {
		return nil, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return nil, err
	}

	return &value, nil
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
