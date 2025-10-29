package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"monserv/internal/notifier"
	srv "monserv/internal/server"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load() // load .env if present
	cfg := srv.LoadConfig()
	log.Printf("Starting monitor: agents=%d poll=%s memTh=%.1f%% diskTh=%.1f%% logThresh=%v", len(cfg.Agents), cfg.PollInterval, cfg.MemThreshold, cfg.DiskThreshold, cfg.LogThresholds)

	baseNotifier := notifier.FromEnv()
	// Add cooldown of 30 minutes to avoid spam
	n := notifier.NewCooldown(baseNotifier, 30*time.Minute)

	p := srv.NewPoller(cfg, n)
	stop := make(chan struct{})
	go p.Start(stop)
	defer close(stop)

	r := gin.Default()
	// Templates
	tmpl := template.Must(template.ParseFiles("web/templates/index.html"))
	r.GET("/", func(c *gin.Context) {
		agents, latest := p.Snapshot()
		data := map[string]any{
			"Agents":  agents,
			"Latest":  latest,
			"Updated": time.Now(),
			"MemTh":   cfg.MemThreshold,
			"DiskTh":  cfg.DiskThreshold,
			"ProcTh":  cfg.ProcThreshold,
		}
		c.Status(http.StatusOK)
		_ = tmpl.Execute(c.Writer, data)
	})
	// API for JSON polling
	r.GET("/api/state", func(c *gin.Context) { _, latest := p.Snapshot(); c.JSON(http.StatusOK, latest) })

	// Serve static
	r.Static("/static", "web/static")

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	_ = r.Run(":" + port)
}
