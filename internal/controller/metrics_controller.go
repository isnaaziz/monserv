package controller

import (
	"net/http"

	"monserv/internal/dto"
	"monserv/internal/service"

	"github.com/gin-gonic/gin"
)

// MetricsController handles HTTP requests for metrics
type MetricsController struct {
	service service.MetricsService
}

func NewMetricsController(service service.MetricsService) *MetricsController {
	return &MetricsController{service: service}
}

// GetAllServers godoc
// @Summary Get all monitored servers
// @Description Get list of all servers being monitored with their current status and metrics
// @Tags Servers
// @Accept json
// @Produce json
// @Success 200 {object} dto.APIResponse{data=dto.ServerListResponse} "Successfully retrieved server list"
// @Failure 500 {object} dto.APIResponse "Internal server error"
// @Router /v1/servers [get]
func (c *MetricsController) GetAllServers(ctx *gin.Context) {
	servers := c.service.GetAllServers()

	ctx.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Message: "Successfully retrieved server list",
		Data:    servers,
	})
}

// GetServerMetrics godoc
// @Summary Get metrics for specific server
// @Description Get detailed metrics for a specific server by URL
// @Tags Servers
// @Accept json
// @Produce json
// @Param url query string true "Server URL" example(ssh://scada:@Grita123@192.168.4.3:2222)
// @Success 200 {object} dto.APIResponse{data=dto.ServerMetricsResponse} "Successfully retrieved server metrics"
// @Failure 400 {object} dto.APIResponse "Bad request - URL parameter missing"
// @Failure 404 {object} dto.APIResponse "Server not found"
// @Failure 500 {object} dto.APIResponse "Internal server error"
// @Router /v1/servers/metrics [get]
func (c *MetricsController) GetServerMetrics(ctx *gin.Context) {
	serverURL := ctx.Query("url")
	if serverURL == "" {
		ctx.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Error:   "server URL is required",
		})
		return
	}

	metrics, err := c.service.GetServerMetrics(serverURL)
	if err != nil {
		ctx.JSON(http.StatusNotFound, dto.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Message: "Successfully retrieved server metrics",
		Data:    metrics,
	})
}

// GetActiveAlerts godoc
// @Summary Get all active alerts
// @Description Get list of all currently active alerts across all servers
// @Tags Alerts
// @Accept json
// @Produce json
// @Success 200 {object} dto.APIResponse{data=[]dto.AlertResponse} "Successfully retrieved active alerts"
// @Failure 500 {object} dto.APIResponse "Internal server error"
// @Router /v1/alerts/active [get]
func (c *MetricsController) GetActiveAlerts(ctx *gin.Context) {
	alerts := c.service.GetActiveAlerts()

	ctx.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Message: "Successfully retrieved active alerts",
		Data:    alerts,
	})
}

// GetServerHealth godoc
// @Summary Get health status of all servers
// @Description Get simplified health status (online/offline/warning/alert) for all servers
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} dto.APIResponse{data=dto.HealthResponse} "Successfully retrieved health status"
// @Failure 500 {object} dto.APIResponse "Internal server error"
// @Router /v1/health [get]
func (c *MetricsController) GetServerHealth(ctx *gin.Context) {
	health := c.service.GetServerHealth()

	ctx.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Message: "Successfully retrieved health status",
		Data:    health,
	})
}

// RegisterRoutes registers all routes for metrics controller
func (c *MetricsController) RegisterRoutes(router *gin.RouterGroup) {
	v1 := router.Group("/v1")
	{
		// Server endpoints
		servers := v1.Group("/servers")
		{
			servers.GET("", c.GetAllServers)
			servers.GET("/metrics", c.GetServerMetrics)
		}

		// Alert endpoints
		alerts := v1.Group("/alerts")
		{
			alerts.GET("/active", c.GetActiveAlerts)
		}

		// Health endpoint
		v1.GET("/health", c.GetServerHealth)
	}
}
