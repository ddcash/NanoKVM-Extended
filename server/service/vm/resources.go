package vm

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"NanoKVM-Server/proto"
)

// Resource usage for the SG2002, which is a single core with roughly 158 MB of
// RAM visible to Linux. Everything here is read from /proc and /sys, so polling
// it costs almost nothing and never touches the video or HID paths.
const (
	statFile     = "/proc/stat"
	memInfoFile  = "/proc/meminfo"
	loadAvgFile  = "/proc/loadavg"
	uptimeFile   = "/proc/uptime"
	thermalFile  = "/sys/class/thermal/thermal_zone0/temp"
	rootMount    = "/"
	minSampleGap = 500 * time.Millisecond
)

// CPU percentage is a delta between two samples of cumulative jiffies, so the
// previous sample has to be remembered between requests.
var (
	cpuMu       sync.Mutex
	lastTotal   uint64
	lastIdle    uint64
	lastSampled time.Time
	lastPercent float64
)

func (s *Service) GetResources(c *gin.Context) {
	var rsp proto.Response

	rsp.OkRspWithData(c, ReadResources())
}

// ReadResources is exported so the Home Assistant bridge can publish the same
// figures the settings tab shows.
func ReadResources() *proto.GetResourcesRsp {
	data := &proto.GetResourcesRsp{
		CpuPercent:  readCPUPercent(),
		Temperature: readTemperature(),
	}

	total, available := readMemory()
	data.MemoryTotal = total
	data.MemoryAvailable = available
	if total > 0 {
		data.MemoryPercent = round1(float64(total-available) / float64(total) * 100)
	}

	diskTotal, diskFree := readDisk(rootMount)
	data.DiskTotal = diskTotal
	data.DiskFree = diskFree
	if diskTotal > 0 {
		data.DiskPercent = round1(float64(diskTotal-diskFree) / float64(diskTotal) * 100)
	}

	data.Load1, data.Load5, data.Load15 = readLoadAvg()
	data.UptimeSeconds = readUptime()

	return data
}

// readCPUPercent samples cumulative jiffies and reports the busy share since
// the previous call. The first call has no baseline and reports 0.
func readCPUPercent() float64 {
	file, err := os.Open(statFile)
	if err != nil {
		return 0
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0
	}

	fields := strings.Fields(scanner.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0
	}

	var total, idle uint64
	for i := 1; i < len(fields); i++ {
		value, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			continue
		}
		total += value
		// Fields 4 and 5 are idle and iowait.
		if i == 4 || i == 5 {
			idle += value
		}
	}

	cpuMu.Lock()
	defer cpuMu.Unlock()

	// Reuse the previous answer when polled faster than the sample gap, rather
	// than dividing by a tiny delta and reporting noise.
	if !lastSampled.IsZero() && time.Since(lastSampled) < minSampleGap {
		return lastPercent
	}

	var percent float64
	if lastTotal != 0 && total > lastTotal {
		totalDelta := total - lastTotal
		idleDelta := idle - lastIdle
		percent = round1(float64(totalDelta-idleDelta) / float64(totalDelta) * 100)
	}

	lastTotal, lastIdle, lastSampled, lastPercent = total, idle, time.Now(), percent
	return percent
}

// readMemory returns total and available bytes. MemAvailable is the meaningful
// figure: MemFree looks alarmingly small here because the kernel uses most of
// the remainder for cache, which it reclaims on demand.
func readMemory() (total uint64, available uint64) {
	file, err := os.Open(memInfoFile)
	if err != nil {
		return 0, 0
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}

		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		value *= 1024 // /proc/meminfo reports kB

		switch fields[0] {
		case "MemTotal:":
			total = value
		case "MemAvailable:":
			available = value
		}
	}

	return total, available
}

func readDisk(path string) (total uint64, free uint64) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		log.Debugf("failed to statfs %s: %s", path, err)
		return 0, 0
	}

	return stat.Blocks * uint64(stat.Bsize), stat.Bavail * uint64(stat.Bsize)
}

func readLoadAvg() (one float64, five float64, fifteen float64) {
	data, err := os.ReadFile(loadAvgFile)
	if err != nil {
		return 0, 0, 0
	}

	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return 0, 0, 0
	}

	one, _ = strconv.ParseFloat(fields[0], 64)
	five, _ = strconv.ParseFloat(fields[1], 64)
	fifteen, _ = strconv.ParseFloat(fields[2], 64)
	return one, five, fifteen
}

func readUptime() int64 {
	data, err := os.ReadFile(uptimeFile)
	if err != nil {
		return 0
	}

	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0
	}

	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	return int64(seconds)
}

// readTemperature reports degrees Celsius, or 0 when the SoC exposes no
// thermal zone.
func readTemperature() float64 {
	data, err := os.ReadFile(thermalFile)
	if err != nil {
		return 0
	}

	milli, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	if err != nil {
		return 0
	}
	return round1(milli / 1000)
}

func round1(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}
