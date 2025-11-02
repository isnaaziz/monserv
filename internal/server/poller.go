package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	m "monserv/internal/metrics"
	"monserv/internal/notifier"
	"monserv/internal/repository"
)

// WebSocketBroadcaster interface untuk broadcast metrics via WebSocket
type WebSocketBroadcaster interface {
	BroadcastMetrics(state map[string]*m.ServerMetrics)
	BroadcastAlert(alertType, subject, message string)
}

type State struct {
	mu     sync.RWMutex
	Agents []string
	Latest map[string]*m.ServerMetrics
	Alerts map[string]bool
}

func NewState(agents []string) *State {
	return &State{Agents: agents, Latest: map[string]*m.ServerMetrics{}, Alerts: map[string]bool{}}
}

// Snapshot safely returns a copy of agents and latest metrics map for read-only use in handlers
func (s *State) Snapshot() ([]string, map[string]*m.ServerMetrics) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	agentsCopy := append([]string(nil), s.Agents...)
	latestCopy := make(map[string]*m.ServerMetrics, len(s.Latest))
	for k, v := range s.Latest {
		latestCopy[k] = v
	}
	return agentsCopy, latestCopy
}

type Poller struct {
	Cfg      Config
	State    *State
	Notifier notifier.Notifier
	Client   *http.Client
	Repo     repository.MetricsRepository // Tambahan untuk sync ke repository
	WSHub    WebSocketBroadcaster         // Tambahan untuk WebSocket broadcast
}

func NewPoller(cfg Config, n notifier.Notifier) *Poller {
	return &Poller{
		Cfg:      cfg,
		State:    NewState(cfg.Agents),
		Notifier: n,
		Client:   &http.Client{Timeout: 5 * time.Second},
		Repo:     nil, // akan diset dari main.go
		WSHub:    nil, // akan diset dari main.go
	}
}

// Snapshot proxies state's snapshot
func (p *Poller) Snapshot() ([]string, map[string]*m.ServerMetrics) { return p.State.Snapshot() }

func (p *Poller) Start(stop <-chan struct{}) {
	ticker := time.NewTicker(p.Cfg.PollInterval)
	defer ticker.Stop()
	p.runOnce()
	for {
		select {
		case <-ticker.C:
			p.runOnce()
		case <-stop:
			return
		}
	}
}

func (p *Poller) runOnce() {
	var wg sync.WaitGroup
	for _, url := range p.Cfg.Agents {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			var met *m.ServerMetrics
			var err error
			if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
				met, err = p.fetchHTTP(u)
			} else if strings.HasPrefix(u, "ssh://") {
				met, err = CollectViaSSH(u)
			} else {
				// Default assume HTTP base
				met, err = p.fetchHTTP(u)
			}
			if err == nil && met != nil {
				p.State.mu.Lock()
				p.State.Latest[u] = met
				p.State.mu.Unlock()

				// Sync ke repository jika tersedia
				if p.Repo != nil {
					p.Repo.Set(u, met)
				}

				p.checkAlerts(u, met)
			}
		}(url)
	}
	wg.Wait()

	// Broadcast metrics update via WebSocket setelah semua agents selesai polling
	if p.WSHub != nil {
		_, latest := p.State.Snapshot()
		p.WSHub.BroadcastMetrics(latest)
	}
}

func (p *Poller) fetchHTTP(base string) (*m.ServerMetrics, error) {
	resp, err := p.Client.Get(fmt.Sprintf("%s/metrics", base))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var out m.ServerMetrics
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (p *Poller) checkAlerts(agentURL string, mtr *m.ServerMetrics) {
	if p.Cfg.LogThresholds {
		log.Printf("[THRESHOLD] host=%s cpu=%.1f%% th=%.1f%%", mtr.Hostname, mtr.CPU.UsedPercent, p.Cfg.CPUThreshold)
		log.Printf("[THRESHOLD] host=%s mem=%.1f%% th=%.1f%%", mtr.Hostname, mtr.Memory.UsedPercent, p.Cfg.MemThreshold)
		for _, d := range mtr.Disks {
			log.Printf("[THRESHOLD] host=%s mount=%s disk=%.1f%% th=%.1f%%", mtr.Hostname, d.Mountpoint, d.UsedPercent, p.Cfg.DiskThreshold)
		}
		for _, pr := range mtr.TopProcsByMem {
			log.Printf("[THRESHOLD] host=%s pid=%d name=%s ram=%.1f%% th=%.1f%%", mtr.Hostname, pr.PID, pr.Name, pr.PercentRAM, p.Cfg.ProcThreshold)
		}
	}
	// CPU threshold
	keyCPU := fmt.Sprintf("%s|cpu", agentURL)
	if mtr.CPU.UsedPercent >= p.Cfg.CPUThreshold {
		p.raiseOnce(keyCPU, fmt.Sprintf("[ALERT] %s CPU high", mtr.Hostname),
			fmt.Sprintf("CPU used %.1f%% (threshold %.1f%%)", mtr.CPU.UsedPercent, p.Cfg.CPUThreshold))
	} else {
		p.recoverIfActive(keyCPU, fmt.Sprintf("[RECOVERED] %s CPU", mtr.Hostname),
			fmt.Sprintf("CPU back to %.1f%%", mtr.CPU.UsedPercent))
	}
	// Memory threshold
	keyMem := fmt.Sprintf("%s|mem", agentURL)
	if mtr.Memory.UsedPercent >= p.Cfg.MemThreshold {
		p.raiseOnce(keyMem, fmt.Sprintf("[ALERT] %s memory high", mtr.Hostname),
			fmt.Sprintf("Memory used %.1f%% (threshold %.1f%%)", mtr.Memory.UsedPercent, p.Cfg.MemThreshold))
	} else {
		p.recoverIfActive(keyMem, fmt.Sprintf("[RECOVERED] %s memory", mtr.Hostname),
			fmt.Sprintf("Memory back to %.1f%%", mtr.Memory.UsedPercent))
	}
	// Disk thresholds
	for _, d := range mtr.Disks {
		key := fmt.Sprintf("%s|disk|%s", agentURL, d.Mountpoint)
		if d.UsedPercent >= p.Cfg.DiskThreshold {
			p.raiseOnce(key, fmt.Sprintf("[ALERT] %s disk %s high", mtr.Hostname, d.Mountpoint),
				fmt.Sprintf("Disk %s used %.1f%% (threshold %.1f%%)", d.Mountpoint, d.UsedPercent, p.Cfg.DiskThreshold))
		} else {
			p.recoverIfActive(key, fmt.Sprintf("[RECOVERED] %s disk %s", mtr.Hostname, d.Mountpoint),
				fmt.Sprintf("Disk %s back to %.1f%%", d.Mountpoint, d.UsedPercent))
		}
	}

	// Processes thresholds
	above := map[string]bool{}
	for _, pr := range mtr.TopProcsByMem {
		k := fmt.Sprintf("%s|proc|%d", agentURL, pr.PID)
		if pr.PercentRAM >= float32(p.Cfg.ProcThreshold) {
			p.raiseOnce(k, fmt.Sprintf("[ALERT] %s proc %s(%d) RAM high", mtr.Hostname, pr.Name, pr.PID),
				fmt.Sprintf("Process %s(%d) uses %.1f%% RAM (th %.1f%%)", pr.Name, pr.PID, pr.PercentRAM, p.Cfg.ProcThreshold))
			above[k] = true
		} else {
			p.recoverIfActive(k, fmt.Sprintf("[RECOVERED] %s proc %s(%d) RAM", mtr.Hostname, pr.Name, pr.PID),
				fmt.Sprintf("Process %s(%d) back to %.1f%% RAM", pr.Name, pr.PID, pr.PercentRAM))
		}
	}

	// Collect keys to recover without holding lock while sending notifications
	var toRecover []string
	p.State.mu.Lock()
	for key := range p.State.Alerts {
		if strings.HasPrefix(key, agentURL+"|proc|") && !above[key] {
			delete(p.State.Alerts, key)
			toRecover = append(toRecover, key)
		}
	}
	p.State.mu.Unlock()

	for _, key := range toRecover {
		log.Printf("[RECOVERED-SEND] %s | %s", "[RECOVERED] process below threshold", key)
		_ = p.Notifier.Send("[RECOVERED] process below threshold", key)
	}
}

func (p *Poller) raiseOnce(key, subject, body string) {
	p.State.mu.Lock()
	active := p.State.Alerts[key]
	if !active {
		p.State.Alerts[key] = true
	}
	p.State.mu.Unlock()
	if !active {
		if p.Repo != nil {
			p.Repo.SetAlert(key, true)
		}
		// Broadcast alert via WebSocket
		if p.WSHub != nil {
			p.WSHub.BroadcastAlert("alert", subject, body)
		}
		log.Printf("[ALERT-SEND] %s | %s", subject, body)
		_ = p.Notifier.Send(subject, body)
	}
}

func (p *Poller) recoverIfActive(key, subject, body string) {
	p.State.mu.Lock()
	active := p.State.Alerts[key]
	if active {
		delete(p.State.Alerts, key)
	}
	p.State.mu.Unlock()
	if active {
		if p.Repo != nil {
			p.Repo.DeleteAlert(key)
		}
		// Broadcast recovery via WebSocket
		if p.WSHub != nil {
			p.WSHub.BroadcastAlert("recovery", subject, body)
		}
		log.Printf("[RECOVERED-SEND] %s | %s", subject, body)
		_ = p.Notifier.Send(subject, body)
	}
}
