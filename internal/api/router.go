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
)

type mqttService interface {
	Enabled() bool
	Connected() bool
	Topics() []string
	Publish(ctx context.Context, topic string, payload []byte, qos byte, retained bool) error
	RecentMessages(limit int) []mqttclient.MessageSnapshot
}

type mqttPublishRequest struct {
	Topic    string          `json:"topic" binding:"required"`
	QoS      uint8           `json:"qos"`
	Retained bool            `json:"retained"`
	Payload  json.RawMessage `json:"payload" binding:"required"`
}

func NewRouter(appLogger *slog.Logger, mqttSvc mqttService) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestLogger(appLogger))

	router.GET("/health", func(c *gin.Context) {
		ok(c, gin.H{
			"status":         "ok",
			"time":           time.Now().UTC(),
			"mqtt_enabled":   mqttSvc.Enabled(),
			"mqtt_connected": mqttSvc.Connected(),
		})
	})

	apiGroup := router.Group("/api")
	{
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
