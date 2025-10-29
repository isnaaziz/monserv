package repository

import (
	"sync"
	"time"

	m "monserv/internal/metrics"
	"monserv/internal/utils"
)

// MetricsRepository interface untuk data access
type MetricsRepository interface {
	GetAll() map[string]*m.ServerMetrics
	Get(serverURL string) (*m.ServerMetrics, bool)
	FindByMaskedURL(maskedURL string) (*m.ServerMetrics, string, bool) // returns metrics, actualURL, found
	Set(serverURL string, metrics *m.ServerMetrics)
	Delete(serverURL string)
	GetServerList() []string
	GetAlerts() map[string]bool
	SetAlert(key string, active bool)
	DeleteAlert(key string)
	GetLastUpdate(serverURL string) time.Time
}

// InMemoryMetricsRepository implementasi in-memory
type InMemoryMetricsRepository struct {
	mu          sync.RWMutex
	metrics     map[string]*m.ServerMetrics
	alerts      map[string]bool
	lastUpdates map[string]time.Time
}

func NewInMemoryMetricsRepository() *InMemoryMetricsRepository {
	return &InMemoryMetricsRepository{
		metrics:     make(map[string]*m.ServerMetrics),
		alerts:      make(map[string]bool),
		lastUpdates: make(map[string]time.Time),
	}
}

func (r *InMemoryMetricsRepository) GetAll() map[string]*m.ServerMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*m.ServerMetrics, len(r.metrics))
	for k, v := range r.metrics {
		result[k] = v
	}
	return result
}

func (r *InMemoryMetricsRepository) Get(serverURL string) (*m.ServerMetrics, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics, ok := r.metrics[serverURL]
	return metrics, ok
}

// FindByMaskedURL finds metrics by masked URL (e.g., ssh://user:***@host:port)
// Returns the metrics, actual URL, and whether it was found
func (r *InMemoryMetricsRepository) FindByMaskedURL(maskedURL string) (*m.ServerMetrics, string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// First try exact match
	if metrics, ok := r.metrics[maskedURL]; ok {
		return metrics, maskedURL, true
	}

	// If not found, try to match by masked version
	for actualURL, metrics := range r.metrics {
		if utils.MaskPassword(actualURL) == maskedURL {
			return metrics, actualURL, true
		}
	}

	return nil, "", false
}

func (r *InMemoryMetricsRepository) Set(serverURL string, metrics *m.ServerMetrics) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.metrics[serverURL] = metrics
	r.lastUpdates[serverURL] = time.Now()
}

func (r *InMemoryMetricsRepository) Delete(serverURL string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.metrics, serverURL)
	delete(r.lastUpdates, serverURL)
}

func (r *InMemoryMetricsRepository) GetServerList() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	servers := make([]string, 0, len(r.metrics))
	for k := range r.metrics {
		servers = append(servers, k)
	}
	return servers
}

func (r *InMemoryMetricsRepository) GetAlerts() map[string]bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]bool, len(r.alerts))
	for k, v := range r.alerts {
		result[k] = v
	}
	return result
}

func (r *InMemoryMetricsRepository) SetAlert(key string, active bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.alerts[key] = active
}

func (r *InMemoryMetricsRepository) DeleteAlert(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.alerts, key)
}

func (r *InMemoryMetricsRepository) GetLastUpdate(serverURL string) time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if t, ok := r.lastUpdates[serverURL]; ok {
		return t
	}
	return time.Time{}
}
