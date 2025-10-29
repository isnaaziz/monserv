package main

import (
	"os"

	"monserv/docs"
)

func init() {
	// Set dynamic host dari environment variable atau default kosong
	// Jika kosong, Swagger UI akan auto-detect dari browser
	apiHost := os.Getenv("API_HOST")
	if apiHost == "" {
		// Kosongkan host agar Swagger auto-detect
		docs.SwaggerInfo.Host = ""
	} else {
		docs.SwaggerInfo.Host = apiHost
	}

	docs.SwaggerInfo.Title = "MonServ API"
	docs.SwaggerInfo.Description = `Server Monitoring System REST API for Mobile Integration

## WebSocket Real-Time Updates
Connect to WebSocket endpoint '/ws' for real-time metrics and alerts.
Example: ws://your-server-host:port/ws
See /docs/WEBSOCKET_API.md for complete documentation.

## Dynamic Host Support
API dapat diakses dari mana saja. Swagger UI akan auto-detect host dari browser.
Set environment variable API_HOST untuk override (optional).

## Deployment
- Development: http://localhost:18904
- Production: Set API_HOST environment variable`

	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.BasePath = "/api"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}
}
