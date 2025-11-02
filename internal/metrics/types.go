package metrics

import "time"

// CPU holds processor usage details
type CPU struct {
	Cores       int     `json:"cores"`
	UsedPercent float64 `json:"usedPercent"`
	ModelName   string  `json:"modelName"`
}

// Memory holds RAM usage details
type Memory struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"usedPercent"`
}

// DiskPartition describes a single mount point usage
type DiskPartition struct {
	Device      string  `json:"device"`
	Mountpoint  string  `json:"mountpoint"`
	Fstype      string  `json:"fstype"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"usedPercent"`
}

// ProcMem describes a process memory usage summary
type ProcMem struct {
	PID        int32   `json:"pid"`
	Name       string  `json:"name"`
	Username   string  `json:"username"`
	RSSBytes   uint64  `json:"rssBytes"`
	PercentRAM float32 `json:"percentRAM"`
	Cmdline    string  `json:"cmdline"`
}

// ServerMetrics is the full payload exposed by agents
type ServerMetrics struct {
	Hostname       string          `json:"hostname"`
	UptimeSeconds  uint64          `json:"uptimeSeconds"`
	CPU            CPU             `json:"cpu"`
	Memory         Memory          `json:"memory"`
	Disks          []DiskPartition `json:"disks"`
	TopProcsByMem  []ProcMem       `json:"topProcsByMem"`
	GeneratedAtUTC time.Time       `json:"generatedAtUtc"`
}
