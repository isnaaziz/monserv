package dto

import (
	m "monserv/internal/metrics"
	"monserv/internal/utils"
	"time"
)

// APIResponse generic wrapper untuk semua response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ServerMetricsResponse untuk response metrics server
type ServerMetricsResponse struct {
	Hostname       string            `json:"hostname" example:"scadanas"`
	UptimeSeconds  uint64            `json:"uptime_seconds" example:"3600"`
	Memory         MemoryResponse    `json:"memory"`
	Disks          []DiskResponse    `json:"disks"`
	TopProcsByMem  []ProcessResponse `json:"top_processes_by_memory"`
	GeneratedAtUTC time.Time         `json:"generated_at_utc" example:"2025-10-29T12:00:00Z"`
}

type MemoryResponse struct {
	Total       uint64  `json:"total_bytes" example:"8589934592"`
	Used        uint64  `json:"used_bytes" example:"4294967296"`
	Free        uint64  `json:"free_bytes" example:"4294967296"`
	UsedPercent float64 `json:"used_percent" example:"50.0"`
}

type DiskResponse struct {
	Device      string  `json:"device" example:"/dev/sda1"`
	Mountpoint  string  `json:"mountpoint" example:"/"`
	Fstype      string  `json:"fstype" example:"ext4"`
	Total       uint64  `json:"total_bytes" example:"107374182400"`
	Used        uint64  `json:"used_bytes" example:"53687091200"`
	Free        uint64  `json:"free_bytes" example:"53687091200"`
	UsedPercent float64 `json:"used_percent" example:"50.0"`
}

type ProcessResponse struct {
	PID        int32   `json:"pid" example:"1234"`
	Name       string  `json:"name" example:"postgres"`
	Username   string  `json:"username" example:"postgres"`
	RSSBytes   uint64  `json:"rss_bytes" example:"536870912"`
	PercentRAM float32 `json:"percent_ram" example:"6.25"`
	Cmdline    string  `json:"cmdline" example:"postgres -D /var/lib/postgresql/data"`
}

// ServerListResponse untuk list semua server
type ServerListResponse struct {
	Servers []ServerStatusResponse `json:"servers"`
	Total   int                    `json:"total" example:"4"`
}

type ServerStatusResponse struct {
	URL        string                 `json:"url" example:"ssh://scada:***@192.168.4.3:2222"`
	Status     string                 `json:"status" example:"online" enums:"online,offline,warning,alert"`
	Metrics    *ServerMetricsResponse `json:"metrics,omitempty"`
	LastUpdate time.Time              `json:"last_update" example:"2025-10-29T12:00:00Z"`
}

// MaskURL masks password in URL for safe display in API responses
func (s *ServerStatusResponse) MaskURL() {
	s.URL = utils.MaskPassword(s.URL)
}

// AlertResponse untuk response alert
type AlertResponse struct {
	ID          string     `json:"id" example:"ssh://scada:***@192.168.4.3:2222|mem"`
	ServerURL   string     `json:"server_url" example:"ssh://scada:***@192.168.4.3:2222"`
	Hostname    string     `json:"hostname" example:"scadanas"`
	Type        string     `json:"type" example:"memory" enums:"memory,disk,process"`
	Severity    string     `json:"severity" example:"critical" enums:"warning,critical"`
	Subject     string     `json:"subject" example:"[ALERT] scadanas memory high"`
	Message     string     `json:"message" example:"Memory used 85.0% (threshold 80.0%)"`
	IsActive    bool       `json:"is_active" example:"true"`
	TriggeredAt time.Time  `json:"triggered_at" example:"2025-10-29T12:00:00Z"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty" example:"2025-10-29T13:00:00Z"`
}

// MaskSensitiveData masks password in URLs for safe display in API responses
func (a *AlertResponse) MaskSensitiveData() {
	a.ID = utils.MaskPassword(a.ID)
	a.ServerURL = utils.MaskPassword(a.ServerURL)
}

// ServerHealthInfo untuk informasi detail setiap server di health check
type ServerHealthInfo struct {
	URL      string `json:"url" example:"ssh://scada:***@192.168.4.3:2222"`
	Hostname string `json:"hostname" example:"scadanas"`
	Status   string `json:"status" example:"online" enums:"online,offline,warning,alert"`
}

// HealthResponse untuk response health check
type HealthResponse struct {
	Status  string                      `json:"status" example:"ok" enums:"ok,degraded,error"`
	Servers map[string]ServerHealthInfo `json:"servers"`
	Total   int                         `json:"total" example:"4"`
	Online  int                         `json:"online" example:"3"`
	Offline int                         `json:"offline" example:"1"`
	Alerts  int                         `json:"alerts" example:"2"`
}

// Converter functions
func ToServerMetricsResponse(m *m.ServerMetrics) *ServerMetricsResponse {
	if m == nil {
		return nil
	}

	disks := make([]DiskResponse, len(m.Disks))
	for i, d := range m.Disks {
		disks[i] = DiskResponse{
			Device:      d.Device,
			Mountpoint:  d.Mountpoint,
			Fstype:      d.Fstype,
			Total:       d.Total,
			Used:        d.Used,
			Free:        d.Free,
			UsedPercent: d.UsedPercent,
		}
	}

	procs := make([]ProcessResponse, len(m.TopProcsByMem))
	for i, p := range m.TopProcsByMem {
		procs[i] = ProcessResponse{
			PID:        p.PID,
			Name:       p.Name,
			Username:   p.Username,
			RSSBytes:   p.RSSBytes,
			PercentRAM: p.PercentRAM,
			Cmdline:    p.Cmdline,
		}
	}

	return &ServerMetricsResponse{
		Hostname:      m.Hostname,
		UptimeSeconds: m.UptimeSeconds,
		Memory: MemoryResponse{
			Total:       m.Memory.Total,
			Used:        m.Memory.Used,
			Free:        m.Memory.Free,
			UsedPercent: m.Memory.UsedPercent,
		},
		Disks:          disks,
		TopProcsByMem:  procs,
		GeneratedAtUTC: m.GeneratedAtUTC,
	}
}
