package handlers

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"panel-backend/internal/models"
	"panel-backend/internal/services"
)

// StatsHandler handles system stats with background collection
type StatsHandler struct {
	pm2    *services.PM2
	mu     sync.RWMutex
	cached *models.Stats
	prevCPU cpuTimes
}

type cpuTimes struct {
	user, nice, system, idle, iowait, irq, softirq, steal uint64
}

// NewStatsHandler creates a new stats handler and starts background collection
func NewStatsHandler(pm2 *services.PM2) *StatsHandler {
	h := &StatsHandler{pm2: pm2}

	// Initial CPU reading
	h.prevCPU = readCPUTimes()

	// Start background collector
	go h.collect()

	return h
}

// Get handles GET /api/stats
func (h *StatsHandler) Get(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	stats := h.cached
	h.mu.RUnlock()

	if stats == nil {
		Success(w, nil)
		return
	}

	Success(w, stats)
}

// collect runs every 10 seconds to gather system stats
func (h *StatsHandler) collect() {
	// Small initial delay to let system stabilize
	time.Sleep(1 * time.Second)
	h.doCollect()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.doCollect()
	}
}

func (h *StatsHandler) doCollect() {
	stats := &models.Stats{}

	// CPU
	current := readCPUTimes()
	stats.CPU = h.calculateCPU(current)
	h.prevCPU = current

	// Memory
	stats.Memory = readMemory()

	// Disk
	stats.Disk = readDisk()

	// System info
	stats.System = readSystem()

	// Apps from PM2
	pm2List, err := h.pm2.List()
	if err == nil {
		running := 0
		stopped := 0
		appList := make([]map[string]interface{}, 0, len(pm2List))
		for _, p := range pm2List {
			if p.Status == "online" {
				running++
			} else {
				stopped++
			}
			appList = append(appList, map[string]interface{}{
				"name":   p.Name,
				"status": p.Status,
				"cpu":    p.CPU,
				"memory": p.Memory,
				"uptime": p.Uptime,
			})
		}
		stats.Apps = models.AppsStats{
			Total:   len(pm2List),
			Running: running,
			Stopped: stopped,
			List:    appList,
		}
	} else {
		stats.Apps = models.AppsStats{List: []interface{}{}}
	}

	h.mu.Lock()
	h.cached = stats
	h.mu.Unlock()
}

func (h *StatsHandler) calculateCPU(current cpuTimes) models.CPUStats {
	prevTotal := h.prevCPU.user + h.prevCPU.nice + h.prevCPU.system + h.prevCPU.idle +
		h.prevCPU.iowait + h.prevCPU.irq + h.prevCPU.softirq + h.prevCPU.steal
	currTotal := current.user + current.nice + current.system + current.idle +
		current.iowait + current.irq + current.softirq + current.steal

	prevIdle := h.prevCPU.idle + h.prevCPU.iowait
	currIdle := current.idle + current.iowait

	totalDelta := float64(currTotal - prevTotal)
	idleDelta := float64(currIdle - prevIdle)

	var usage float64
	if totalDelta > 0 {
		usage = ((totalDelta - idleDelta) / totalDelta) * 100
	}

	// Read CPU model
	model := readCPUModel()

	// Read load average
	loadAvg := readLoadAvg()

	return models.CPUStats{
		Usage:   round2(usage),
		Cores:   runtime.NumCPU(),
		Model:   model,
		LoadAvg: loadAvg,
	}
}

// readCPUTimes reads /proc/stat for CPU times
func readCPUTimes() cpuTimes {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return cpuTimes{}
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) >= 9 {
				return cpuTimes{
					user:    parseUint(fields[1]),
					nice:    parseUint(fields[2]),
					system:  parseUint(fields[3]),
					idle:    parseUint(fields[4]),
					iowait:  parseUint(fields[5]),
					irq:     parseUint(fields[6]),
					softirq: parseUint(fields[7]),
					steal:   parseUint(fields[8]),
				}
			}
		}
	}
	return cpuTimes{}
}

// readMemory reads /proc/meminfo
func readMemory() models.MemoryStats {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return models.MemoryStats{}
	}
	defer f.Close()

	memInfo := make(map[string]int64)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		valStr := strings.TrimSpace(parts[1])
		valStr = strings.TrimSuffix(valStr, " kB")
		valStr = strings.TrimSpace(valStr)
		if val, err := strconv.ParseInt(valStr, 10, 64); err == nil {
			memInfo[key] = val * 1024 // Convert kB to bytes
		}
	}

	total := memInfo["MemTotal"]
	free := memInfo["MemFree"]
	buffers := memInfo["Buffers"]
	cached := memInfo["Cached"]
	sReclaimable := memInfo["SReclaimable"]

	// Used = Total - Free - Buffers - Cached - SReclaimable (same as htop)
	used := total - free - buffers - cached - sReclaimable
	if used < 0 {
		used = total - free
	}

	var percent float64
	if total > 0 {
		percent = float64(used) / float64(total) * 100
	}

	return models.MemoryStats{
		Total:   total,
		Used:    used,
		Free:    total - used,
		Percent: round2(percent),
	}
}

// readDisk reads disk usage for root filesystem
func readDisk() models.DiskStats {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return models.DiskStats{}
	}

	total := int64(stat.Blocks) * int64(stat.Bsize)
	free := int64(stat.Bavail) * int64(stat.Bsize)
	used := total - free

	var percent float64
	if total > 0 {
		percent = float64(used) / float64(total) * 100
	}

	return models.DiskStats{
		Total:   total,
		Used:    used,
		Percent: round2(percent),
	}
}

// readSystem reads system information
func readSystem() models.SystemStats {
	hostname, _ := os.Hostname()

	// Read uptime from /proc/uptime
	uptimeStr := ""
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 0 {
			if secs, err := strconv.ParseFloat(fields[0], 64); err == nil {
				uptimeStr = formatUptime(int(secs))
			}
		}
	}

	return models.SystemStats{
		Uptime:   uptimeStr,
		Hostname: hostname,
		Platform: runtime.GOOS,
		Arch:     runtime.GOARCH,
	}
}

// readCPUModel reads the CPU model name
func readCPUModel() string {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "Unknown"
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "Unknown"
}

// readLoadAvg reads /proc/loadavg
func readLoadAvg() []float64 {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return []float64{0, 0, 0}
	}

	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return []float64{0, 0, 0}
	}

	result := make([]float64, 3)
	for i := 0; i < 3; i++ {
		result[i], _ = strconv.ParseFloat(fields[i], 64)
	}
	return result
}

func formatUptime(totalSecs int) string {
	days := totalSecs / 86400
	hours := (totalSecs % 86400) / 3600
	mins := (totalSecs % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

func parseUint(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}

func round2(f float64) float64 {
	return float64(int(f*100)) / 100
}
