package server

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	m "monserv/internal/metrics"

	"golang.org/x/crypto/ssh"
)

// CollectViaSSH connects to an SSH target specified as ssh://user:pass@host:port and gathers metrics
func CollectViaSSH(target string) (*m.ServerMetrics, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "ssh" {
		return nil, fmt.Errorf("not ssh scheme: %s", u.Scheme)
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "22"
	}
	user := u.User.Username()
	pass, _ := u.User.Password()
	if host == "" || user == "" {
		return nil, errors.New("invalid ssh target; need user@host")
	}

	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(pass)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	conn, err := ssh.Dial("tcp", net.JoinHostPort(host, port), cfg)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	run := func(cmd string) (string, error) {
		s, err := conn.NewSession()
		if err != nil {
			return "", err
		}
		defer s.Close()
		out, err := s.CombinedOutput(cmd)
		return string(out), err
	}

	// CPU usage via top command (1 iteration, 1 second delay)
	cpuOut, _ := run("top -bn2 -d 0.5 | grep '^%Cpu' | tail -n 1")
	cpuUsage := 0.0
	cpuCores := 1
	cpuModel := "unknown"
	
	// Parse CPU usage from top output: %Cpu(s):  5.2 us,  2.1 sy,  0.0 ni, 92.4 id, ...
	if cpuOut != "" {
		fields := strings.Fields(cpuOut)
		if len(fields) >= 8 {
			// id (idle) is usually at index 7
			idleStr := strings.TrimSuffix(fields[7], ",")
			if idle, err := strconv.ParseFloat(idleStr, 64); err == nil {
				cpuUsage = 100.0 - idle
			}
		}
	}
	
	// Get CPU cores count
	coresOut, _ := run("nproc")
	if coresOut != "" {
		if cores, err := strconv.Atoi(strings.TrimSpace(coresOut)); err == nil {
			cpuCores = cores
		}
	}
	
	// Get CPU model
	modelOut, _ := run("cat /proc/cpuinfo | grep 'model name' | head -n 1")
	if modelOut != "" {
		parts := strings.SplitN(modelOut, ":", 2)
		if len(parts) == 2 {
			cpuModel = strings.TrimSpace(parts[1])
		}
	}
	
	cpu := m.CPU{
		Cores:       cpuCores,
		UsedPercent: cpuUsage,
		ModelName:   cpuModel,
	}

	// Memory via /proc/meminfo
	memInfo, _ := run("cat /proc/meminfo | egrep 'MemTotal|MemAvailable|MemFree' ")
	var total, available uint64
	scanner := bufio.NewScanner(strings.NewReader(memInfo))
	for scanner.Scan() {
		line := scanner.Text()
		// Example: MemTotal:       16367480 kB
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			val, _ := strconv.ParseUint(fields[1], 10, 64)
			// convert kB to bytes
			valBytes := val * 1024
			if strings.HasPrefix(line, "MemTotal:") {
				total = valBytes
			}
			if strings.HasPrefix(line, "MemAvailable:") {
				available = valBytes
			}
			// memfree not used; available preferred for modern kernels
		}
	}
	used := total - available
	usedPercent := 0.0
	if total > 0 {
		usedPercent = (float64(used) / float64(total)) * 100
	}

	memory := m.Memory{Total: total, Used: used, Free: available, UsedPercent: usedPercent}

	// Disks via df -P -B1
	dfOut, _ := run("df -P -B1 | tail -n +2")
	disks := []m.DiskPartition{}
	s2 := bufio.NewScanner(strings.NewReader(dfOut))
	for s2.Scan() {
		line := s2.Text()
		// Filesystem 1B-blocks Used Available Use% Mounted on
		f := strings.Fields(line)
		if len(f) < 6 {
			continue
		}
		device := f[0]
		totalB, _ := strconv.ParseUint(f[1], 10, 64)
		usedB, _ := strconv.ParseUint(f[2], 10, 64)
		availB, _ := strconv.ParseUint(f[3], 10, 64)
		usedPctStr := f[4]
		usedPctStr = strings.TrimSuffix(usedPctStr, "%")
		usedPct, _ := strconv.ParseFloat(usedPctStr, 64)
		mount := f[5]
		disks = append(disks, m.DiskPartition{Device: device, Mountpoint: mount, Fstype: "", Total: totalB, Used: usedB, Free: availB, UsedPercent: usedPct})
	}

	// Top processes via ps
	psOut, _ := run("ps -eo pid,comm,user,rss --no-headers | sort -k4 -nr | head -n 5")
	procs := []m.ProcMem{}
	s3 := bufio.NewScanner(strings.NewReader(psOut))
	for s3.Scan() {
		f := strings.Fields(s3.Text())
		if len(f) < 4 {
			continue
		}
		pid64, _ := strconv.ParseInt(f[0], 10, 32)
		name := f[1]
		user := f[2]
		rssKB, _ := strconv.ParseUint(f[3], 10, 64)
		rssB := rssKB * 1024
		percent := float32(0)
		if total > 0 {
			percent = float32((float64(rssB) / float64(total)) * 100)
		}
		procs = append(procs, m.ProcMem{PID: int32(pid64), Name: name, Username: user, RSSBytes: rssB, PercentRAM: percent, Cmdline: ""})
	}

	// Hostname
	hn, _ := run("hostname")
	hostname := strings.TrimSpace(hn)

	return &m.ServerMetrics{
		Hostname:       hostname,
		UptimeSeconds:  0,
		CPU:            cpu,
		Memory:         memory,
		Disks:          disks,
		TopProcsByMem:  procs,
		GeneratedAtUTC: time.Now().UTC(),
	}, nil
}
