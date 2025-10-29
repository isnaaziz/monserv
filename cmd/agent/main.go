package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"monserv/internal/agent"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	r := gin.Default()

	coll := agent.NewCollector()

	r.GET("/health", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.GET("/metrics", func(c *gin.Context) {
		ctx := c.Request.Context()
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) < time.Second {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
		}
		m, err := coll.Collect(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, m)
	})

	port := os.Getenv("AGENT_PORT")
	if port == "" {
		port = "9123"
	}
	_ = r.Run(":" + port)
}
