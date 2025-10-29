package service

import (
	"fmt"
	"strings"
	"time"

	"monserv/internal/dto"
	"monserv/internal/repository"
	"monserv/internal/utils"
)

// MetricsService interface untuk business logic
type MetricsService interface {
	GetAllServers() *dto.ServerListResponse
	GetServerMetrics(serverURL string) (*dto.ServerMetricsResponse, error)
	GetActiveAlerts() []dto.AlertResponse
	GetServerHealth() *dto.HealthResponse
}

type metricsService struct {
	repo          repository.MetricsRepository
	memThreshold  float64
	diskThreshold float64
	procThreshold float64
	alertTimeout  time.Duration
}

func NewMetricsService(
	repo repository.MetricsRepository,
	memTh, diskTh, procTh float64,
	alertTimeout time.Duration,
) MetricsService {
	return &metricsService{
		repo:          repo,
		memThreshold:  memTh,
		diskThreshold: diskTh,
		procThreshold: procTh,
		alertTimeout:  alertTimeout,
	}
}

func (s *metricsService) GetAllServers() *dto.ServerListResponse {
	all := s.repo.GetAll()

	servers := make([]dto.ServerStatusResponse, 0, len(all))
	for url, metrics := range all {
		status := s.determineStatus(url, metrics)
		lastUpdate := s.repo.GetLastUpdate(url)
		if lastUpdate.IsZero() && metrics != nil {
			lastUpdate = metrics.GeneratedAtUTC
		}

		serverResp := dto.ServerStatusResponse{
			URL:        url,
			Status:     status,
			Metrics:    dto.ToServerMetricsResponse(metrics),
			LastUpdate: lastUpdate,
		}
		// Mask password in URL before sending to API
		serverResp.MaskURL()

		servers = append(servers, serverResp)
	}

	return &dto.ServerListResponse{
		Servers: servers,
		Total:   len(servers),
	}
}

func (s *metricsService) GetServerMetrics(serverURL string) (*dto.ServerMetricsResponse, error) {
	// Try exact match first
	metrics, ok := s.repo.Get(serverURL)
	if ok {
		return dto.ToServerMetricsResponse(metrics), nil
	}

	// If not found, try to find by masked URL
	metrics, actualURL, found := s.repo.FindByMaskedURL(serverURL)
	if !found {
		return nil, fmt.Errorf("server not found: %s", serverURL)
	}

	// Log for debugging
	if actualURL != serverURL {
		// User provided masked URL, we found it by matching
		_ = actualURL // Just to use the variable
	}

	return dto.ToServerMetricsResponse(metrics), nil
}

func (s *metricsService) GetActiveAlerts() []dto.AlertResponse {
	alerts := s.repo.GetAlerts()
	all := s.repo.GetAll()

	result := make([]dto.AlertResponse, 0, len(alerts))
	for key, active := range alerts {
		if !active {
			continue
		}

		parts := strings.Split(key, "|")
		if len(parts) < 2 {
			continue
		}

		serverURL := parts[0]
		alertType := parts[1]

		metrics, ok := all[serverURL]
		if !ok {
			continue
		}

		alert := dto.AlertResponse{
			ID:          key,
			ServerURL:   serverURL,
			Hostname:    metrics.Hostname,
			Type:        alertType,
			IsActive:    active,
			TriggeredAt: metrics.GeneratedAtUTC,
		}

		// Determine severity and message based on type
		switch alertType {
		case "mem":
			if metrics.Memory.UsedPercent >= s.memThreshold {
				alert.Severity = "critical"
				alert.Subject = fmt.Sprintf("[ALERT] %s memory high", metrics.Hostname)
				alert.Message = fmt.Sprintf("Memory used %.1f%% (threshold %.1f%%)",
					metrics.Memory.UsedPercent, s.memThreshold)
			}
		case "disk":
			for _, disk := range metrics.Disks {
				if disk.UsedPercent >= s.diskThreshold {
					alert.Severity = "critical"
					alert.Subject = fmt.Sprintf("[ALERT] %s disk %s high", metrics.Hostname, disk.Mountpoint)
					alert.Message = fmt.Sprintf("Disk %s used %.1f%% (threshold %.1f%%)",
						disk.Mountpoint, disk.UsedPercent, s.diskThreshold)
					break
				}
			}
		case "proc":
			if len(parts) >= 3 {
				// Process-specific alert
				for _, proc := range metrics.TopProcsByMem {
					if float64(proc.PercentRAM) >= s.procThreshold {
						alert.Severity = "warning"
						alert.Subject = fmt.Sprintf("[ALERT] %s proc %s high", metrics.Hostname, proc.Name)
						alert.Message = fmt.Sprintf("Process %s (PID %d) uses %.1f%% RAM (threshold %.1f%%)",
							proc.Name, proc.PID, proc.PercentRAM, s.procThreshold)
						break
					}
				}
			}
		}

		// Mask password in URL before sending to API
		alert.MaskSensitiveData()

		result = append(result, alert)
	}

	return result
}

func (s *metricsService) GetServerHealth() *dto.HealthResponse {
	all := s.repo.GetAll()
	alerts := s.repo.GetAlerts()

	health := make(map[string]dto.ServerHealthInfo, len(all))
	online := 0
	offline := 0

	for url, metrics := range all {
		status := s.determineStatus(url, metrics)

		// Mask password in URL for health response
		maskedURL := utils.MaskPassword(url)

		// Extract hostname from metrics
		hostname := "unknown"
		if metrics != nil {
			hostname = metrics.Hostname
		}

		health[maskedURL] = dto.ServerHealthInfo{
			URL:      maskedURL,
			Hostname: hostname,
			Status:   status,
		}

		if status == "online" || status == "warning" || status == "alert" {
			online++
		} else {
			offline++
		}
	}

	activeAlerts := 0
	for _, active := range alerts {
		if active {
			activeAlerts++
		}
	}

	overallStatus := "ok"
	if offline > 0 || activeAlerts > 0 {
		overallStatus = "degraded"
	}
	if offline == len(all) {
		overallStatus = "error"
	}

	return &dto.HealthResponse{
		Status:  overallStatus,
		Servers: health,
		Total:   len(all),
		Online:  online,
		Offline: offline,
		Alerts:  activeAlerts,
	}
}

func (s *metricsService) determineStatus(url string, metrics interface{}) string {
	if metrics == nil {
		return "offline"
	}

	// Check if there are active alerts for this server
	alerts := s.repo.GetAlerts()
	hasAlert := false
	hasWarning := false

	for key, active := range alerts {
		if !active {
			continue
		}
		if strings.HasPrefix(key, url+"|") {
			parts := strings.Split(key, "|")
			if len(parts) >= 2 {
				alertType := parts[1]
				if alertType == "mem" || alertType == "disk" {
					hasAlert = true
				} else if alertType == "proc" {
					hasWarning = true
				}
			}
		}
	}

	if hasAlert {
		return "alert"
	}
	if hasWarning {
		return "warning"
	}
	return "online"
}
