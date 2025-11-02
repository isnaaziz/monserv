package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	_ "monserv/docs"
	"monserv/internal/controller"
	"monserv/internal/notifier"
	"monserv/internal/repository"
	srv "monserv/internal/server"
	"monserv/internal/service"
	"monserv/internal/utils"
	ws "monserv/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title MonServ API
// @version 1.0
// @description Server Monitoring System REST API for Mobile Integration
// @description
// @description ## WebSocket Real-Time Updates
// @description Connect to WebSocket endpoint `/ws` for real-time metrics and alerts.
// @description Example: `ws://your-server-host:port/ws`
// @description See `/docs/WEBSOCKET_API.md` for complete documentation.
// @description
// @description ## Dynamic Host Support
// @description API dapat diakses dari mana saja. Swagger UI akan auto-detect host.
// @description Set environment variable `API_HOST` untuk override (optional).
// @contact.name API Support
// @contact.email support@monserv.local
// @BasePath /api
func main() {
	_ = godotenv.Load() // load .env if present
	cfg := srv.LoadConfig()

	// Mask passwords in agent URLs for safe logging
	maskedAgents := utils.MaskPasswords(cfg.Agents)
	log.Printf("Starting monitor: agents=%d poll=%s memTh=%.1f%% diskTh=%.1f%% logThresh=%v", len(maskedAgents), cfg.PollInterval, cfg.MemThreshold, cfg.DiskThreshold, cfg.LogThresholds)
	for i, agent := range maskedAgents {
		log.Printf("  [%d] %s", i+1, agent)
	}

	baseNotifier := notifier.FromEnv()
	n := notifier.NewCooldown(baseNotifier, 30*time.Minute)

	// Setup repository untuk menyimpan metrics
	repo := repository.NewInMemoryMetricsRepository()

	// Setup WebSocket hub
	hub := ws.NewHub()
	go hub.Run()
	log.Printf("WebSocket hub started")

	// Setup service layer
	metricsService := service.NewMetricsService(
		repo,
		cfg.MemThreshold,
		cfg.DiskThreshold,
		cfg.ProcThreshold,
		5*time.Minute,
	)

	p := srv.NewPoller(cfg, n)
	p.Repo = repo
	p.WSHub = hub
	stop := make(chan struct{})
	go p.Start(stop)
	defer close(stop)

	r := gin.Default()
	swaggerURL := ginSwagger.URL("/swagger/doc.json")
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, swaggerURL))

	r.GET("/ws", func(c *gin.Context) {
		ws.ServeWs(hub, c.Writer, c.Request)
	})

	apiGroup := r.Group("/api")
	metricsController := controller.NewMetricsController(metricsService)
	metricsController.RegisterRoutes(apiGroup)

	tmpl := template.Must(template.ParseFiles("web/templates/index.html"))
	r.GET("/", func(c *gin.Context) {
		agents, latest := p.Snapshot()
		data := map[string]any{
			"Agents":  agents,
			"Latest":  latest,
			"Updated": time.Now(),
			"CPUTh":   cfg.CPUThreshold,
			"MemTh":   cfg.MemThreshold,
			"DiskTh":  cfg.DiskThreshold,
			"ProcTh":  cfg.ProcThreshold,
		}
		c.Status(http.StatusOK)
		_ = tmpl.Execute(c.Writer, data)
	})

	// API for JSON polling (legacy, untuk web UI lama)
	r.GET("/api/state", func(c *gin.Context) { _, latest := p.Snapshot(); c.JSON(http.StatusOK, latest) })

	// Test WebSocket page
	r.GET("/test-websocket", func(c *gin.Context) {
		c.File("web/static/test-websocket.html")
	})

	// Test alert endpoint (for debugging)
	r.POST("/api/test-alert", func(c *gin.Context) {
		var req struct {
			AlertType string `json:"alert_type" binding:"required"` // "alert" or "recovery"
			Subject   string `json:"subject" binding:"required"`
			Message   string `json:"message" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Broadcast test alert via WebSocket
		hub.BroadcastAlert(req.AlertType, req.Subject, req.Message)
		log.Printf("[TEST-ALERT] Broadcasted: type=%s subject=%s message=%s", req.AlertType, req.Subject, req.Message)

		c.JSON(http.StatusOK, gin.H{
			"status":  "sent",
			"type":    req.AlertType,
			"subject": req.Subject,
			"message": req.Message,
		})
	})

	// Serve static
	r.Static("/static", "web/static")

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	api_host := os.Getenv("API_HOST")
	if api_host != "" {
		log.Printf("API Host overridden to %s", api_host)
		swaggerURL = ginSwagger.URL("http://" + api_host + "/swagger/doc.json")
	}

	log.Printf("Server starting on :%s", port)
	log.Printf("Swagger UI available at http://%s/swagger/index.html", api_host)
	log.Printf("WebSocket endpoint at ws://%s/ws", api_host)
	_ = r.Run(":" + port)
}
