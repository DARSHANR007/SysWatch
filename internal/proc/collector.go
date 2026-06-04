package proc

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type Process struct {
	PID  int
	Name string
	CPU  float64 // percent
	RSS  int64   // bytes
}

type procStat struct {
	totalJiffies uint64
	procJiffies  uint64
}

type Collector struct {
	mu   sync.Mutex
	prev map[int]procStat
}

func NewCollector() *Collector {
	return &Collector{prev: make(map[int]procStat)}
}

// readTotalJiffies reads first line of /proc/stat and sums the cpu fields
func readTotalJiffies() (uint64, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return 0, fmt.Errorf("empty /proc/stat")
	}
	parts := strings.Fields(scanner.Text())
	var total uint64
	for _, p := range parts[1:] {
		v, _ := strconv.ParseUint(p, 10, 64)
		total += v
	}
	return total, nil
}

// Collect reads /proc and returns a snapshot of processes with simple CPU calc
func (c *Collector) Collect() ([]Process, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	total, err := readTotalJiffies()
	if err != nil {
		return nil, err
	}

	entries, err := ioutil.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	procs := make([]Process, 0, 128)
	seen := make(map[int]struct{})

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		seen[pid] = struct{}{}

		statPath := filepath.Join("/proc", e.Name(), "stat")
		data, err := ioutil.ReadFile(statPath)
		if err != nil {
			continue
		}
		// stat format: pid (comm) state ... utime stime ... rss ...
		// comm can contain spaces; find closing paren
		b := string(data)
		l := strings.Index(b, "(")
		r := strings.LastIndex(b, ")")
		if l < 0 || r < 0 || r <= l {
			continue
		}
		comm := b[l+1 : r]
		fields := strings.Fields(b[r+1:])
		if len(fields) < 20 {
			continue
		}
		// utime is field 13, stime 14 relative to pid,comm,state: as we sliced after )
		utime, _ := strconv.ParseUint(fields[11], 10, 64)
		stime, _ := strconv.ParseUint(fields[12], 10, 64)
		rssPages, _ := strconv.ParseInt(fields[21], 10, 64)

		// page size
		pageSize := int64(os.Getpagesize())
		rss := rssPages * pageSize

		procJ := utime + stime

		prev, ok := c.prev[pid]
		var cpuPercent float64
		if ok {
			dProc := procJ - prev.procJiffies
			dTotal := total - prev.totalJiffies
			if dTotal > 0 {
				cpuPercent = (float64(dProc) / float64(dTotal)) * 100.0
			}
		}

		c.prev[pid] = procStat{totalJiffies: total, procJiffies: procJ}

		procs = append(procs, Process{PID: pid, Name: comm, CPU: cpuPercent, RSS: rss})
	}

	// cleanup prev entries for processes that vanished
	for pid := range c.prev {
		if _, ok := seen[pid]; !ok {
			delete(c.prev, pid)
		}
	}

	return procs, nil
}
