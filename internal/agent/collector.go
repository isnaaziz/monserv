package agent

import (
	"context"
	"sort"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"

	m "monserv/internal/metrics"
)

type Collector struct {
	TopNProcs int
	Timeout   time.Duration
}

func NewCollector() *Collector {
	return &Collector{TopNProcs: 5, Timeout: 5 * time.Second}
}

// Collect gathers metrics for the current host
func (c *Collector) Collect(ctx context.Context) (*m.ServerMetrics, error) {
	// Add timeout
	if _, ok := ctx.Deadline(); !ok && c.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}

	hv, _ := host.InfoWithContext(ctx)

	vm, _ := mem.VirtualMemoryWithContext(ctx)
	memory := m.Memory{}
	if vm != nil {
		memory = m.Memory{
			Total:       vm.Total,
			Used:        vm.Used,
			Free:        vm.Available,
			UsedPercent: vm.UsedPercent,
		}
	}

	parts := []m.DiskPartition{}
	// Use disk.Partitions to get mount points then disk.Usage per mount
	p, _ := disk.PartitionsWithContext(ctx, true)
	for _, part := range p {
		// Skip some virtual or unusual filesystems optionally
		usage, err := disk.UsageWithContext(ctx, part.Mountpoint)
		if err != nil || usage == nil {
			continue
		}
		parts = append(parts, m.DiskPartition{
			Device:      part.Device,
			Mountpoint:  part.Mountpoint,
			Fstype:      part.Fstype,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
		})
	}

	// Processes by memory usage
	procsByMem := []m.ProcMem{}
	procs, _ := process.ProcessesWithContext(ctx)
	for _, p := range procs {
		// Best-effort; ignore errors to avoid heavy failures
		memInfo, err := p.MemoryInfoWithContext(ctx)
		if err != nil || memInfo == nil {
			continue
		}
		percent, _ := p.MemoryPercentWithContext(ctx)
		name, _ := p.NameWithContext(ctx)
		user, _ := p.UsernameWithContext(ctx)
		cmd, _ := p.CmdlineWithContext(ctx)
		procsByMem = append(procsByMem, m.ProcMem{
			PID:        p.Pid,
			Name:       name,
			Username:   user,
			RSSBytes:   memInfo.RSS,
			PercentRAM: percent,
			Cmdline:    cmd,
		})
	}
	sort.Slice(procsByMem, func(i, j int) bool { return procsByMem[i].RSSBytes > procsByMem[j].RSSBytes })
	if c.TopNProcs > 0 && len(procsByMem) > c.TopNProcs {
		procsByMem = procsByMem[:c.TopNProcs]
	}

	return &m.ServerMetrics{
		Hostname:       firstNonEmpty(hv.Hostname, "unknown"),
		UptimeSeconds:  uptime(hv),
		Memory:         memory,
		Disks:          parts,
		TopProcsByMem:  procsByMem,
		GeneratedAtUTC: time.Now().UTC(),
	}, nil
}

func uptime(h *host.InfoStat) uint64 {
	if h == nil {
		return 0
	}
	return h.Uptime
}

func firstNonEmpty(s, d string) string {
	if s != "" {
		return s
	}
	return d
}
