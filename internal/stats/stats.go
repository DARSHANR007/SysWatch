package stats

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type SystemStats struct {
	CPUPercent    float64       // 0-100
	MemoryPercent float64       // 0-100
	DiskPercent   float64       // 0-100
	Uptime        time.Duration // system uptime
	LoadAvg       [3]float64    // 1m, 5m, 15m
	TotalMem      uint64        // bytes
	UsedMem       uint64        // bytes
}

// Collect gathers system stats
func Collect() (SystemStats, error) {
	stats := SystemStats{}

	// CPU percent (simplified: average from /proc/stat)
	cpu, err := getCPUPercent()
	if err == nil {
		stats.CPUPercent = cpu
	}

	// Memory
	mem, used, err := getMemory()
	if err == nil {
		stats.TotalMem = mem
		stats.UsedMem = used
		stats.MemoryPercent = (float64(used) / float64(mem)) * 100.0
	}

	// Disk
	diskPercent, err := getDiskPercent()
	if err == nil {
		stats.DiskPercent = diskPercent
	}

	// Uptime
	uptime, err := getUptime()
	if err == nil {
		stats.Uptime = uptime
	}

	// Load average
	loadAvg, err := getLoadAverage()
	if err == nil {
		stats.LoadAvg = loadAvg
	}

	return stats, nil
}

// getCPUPercent estimates CPU usage (very basic)
func getCPUPercent() (float64, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 5 {
			return 0, fmt.Errorf("unexpected /proc/stat format")
		}

		user, _ := strconv.ParseFloat(fields[1], 64)
		nice, _ := strconv.ParseFloat(fields[2], 64)
		system, _ := strconv.ParseFloat(fields[3], 64)
		idle, _ := strconv.ParseFloat(fields[4], 64)

		total := user + nice + system + idle
		if total == 0 {
			return 0, nil
		}
		used := user + nice + system
		return (used / total) * 100.0, nil
	}
	return 0, fmt.Errorf("no cpu line in /proc/stat")
}

// getMemory returns total and used memory in bytes
func getMemory() (uint64, uint64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	var total, available uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				total, _ = strconv.ParseUint(fields[1], 10, 64)
				total *= 1024 // convert KB to bytes
			}
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				available, _ = strconv.ParseUint(fields[1], 10, 64)
				available *= 1024 // convert KB to bytes
			}
		}
	}
	used := total - available
	return total, used, nil
}

// getDiskPercent returns root partition usage percentage
func getDiskPercent() (float64, error) {
	// Use 'df' output or read from /proc (simplified: just return placeholder)
	// For now, we'll try to parse 'df /' output
	// This is a simplified version; in production use syscall.Statfs
	return 50.0, nil // placeholder
}

// getUptime returns system uptime
func getUptime() (time.Duration, error) {
	file, err := os.Open("/proc/uptime")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 1 {
			seconds, err := strconv.ParseFloat(fields[0], 64)
			if err == nil {
				return time.Duration(seconds) * time.Second, nil
			}
		}
	}
	return 0, fmt.Errorf("could not parse uptime")
}

// getLoadAverage returns 1m, 5m, 15m load averages
func getLoadAverage() ([3]float64, error) {
	file, err := os.Open("/proc/loadavg")
	if err != nil {
		return [3]float64{}, err
	}
	defer file.Close()

	var loadAvg [3]float64
	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			loadAvg[0], _ = strconv.ParseFloat(fields[0], 64)
			loadAvg[1], _ = strconv.ParseFloat(fields[1], 64)
			loadAvg[2], _ = strconv.ParseFloat(fields[2], 64)
		}
	}
	return loadAvg, nil
}

// FormatUptime returns a human-readable uptime string
func FormatUptime(d time.Duration) string {
	days := d / (24 * time.Hour)
	d = d % (24 * time.Hour)
	hours := d / time.Hour
	d = d % time.Hour
	minutes := d / time.Minute

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
