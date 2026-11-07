package sysinfo

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var tagCPU = []byte("cpu ")

// ParseProcStat parses the aggregate CPU line from /proc/stat content.
// It computes total and idle jiffies and returns instantaneous utilization
// compared to prevTotal and prevIdle.
// Guarantees zero heap allocations.
func ParseProcStat(data []byte, prevTotal, prevIdle uint64) (total, idle uint64, usagePct float64, err error) {
	d := data
	if !bytes.HasPrefix(d, tagCPU) {
		idx := bytes.Index(d, []byte("\ncpu "))
		if idx < 0 {
			return 0, 0, 0, errors.New("no aggregate cpu line found in /proc/stat")
		}
		d = d[idx+1:]
	}
	d = d[len(tagCPU):]

	// Read up to 10 CPU timing fields:
	// user, nice, system, idle, iowait, irq, softirq, steal, guest, guest_nice
	var fields [10]uint64
	for i := 0; i < 10 && len(d) > 0; i++ {
		val, n := ParseUint(d)
		fields[i] = val
		d = d[n:]
	}

	for i := 0; i < 10; i++ {
		total += fields[i]
	}
	// Idle time is idle (field 3) + iowait (field 4)
	idle = fields[3] + fields[4]

	if prevTotal > 0 && total >= prevTotal {
		dTotal := total - prevTotal
		dIdle := idle - prevIdle
		if dTotal > 0 {
			if dIdle > dTotal {
				dIdle = dTotal
			}
			usagePct = float64(dTotal-dIdle) / float64(dTotal) * 100.0
			if usagePct > 100.0 {
				usagePct = 100.0
			} else if usagePct < 0.0 {
				usagePct = 0.0
			}
		}
	}
	return total, idle, usagePct, nil
}

// CPUSampler samples aggregate CPU usage, frequency, temperature, and RAPL power
// using pre-opened file descriptors and stack buffers for zero heap allocations.
type CPUSampler struct {
	statFile       *os.File
	freqFile       *os.File
	tempFile       *os.File
	raplFile       *os.File
	buf            [2048]byte
	smallBuf       [64]byte
	prevTotal      uint64
	prevIdle       uint64
	prevEnergyUJ   uint64
	prevEnergyTime time.Time
	modelName      string
}

// NewCPUSampler discovers hardware paths and opens file descriptors.
func NewCPUSampler() (*CPUSampler, error) {
	statF, err := os.Open("/proc/stat")
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/stat: %w", err)
	}

	s := &CPUSampler{
		statFile:  statF,
		modelName: readCPUModelName(),
	}

	// 1. Frequency source
	freqPaths := []string{
		"/sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq",
		"/sys/devices/system/cpu/cpufreq/policy0/scaling_cur_freq",
	}
	for _, p := range freqPaths {
		if f, err := os.Open(p); err == nil {
			s.freqFile = f
			break
		}
	}

	// 2. Temperature source
	tempPath := findCPUThermalPath()
	if tempPath != "" {
		if f, err := os.Open(tempPath); err == nil {
			s.tempFile = f
		}
	}

	// 3. RAPL Energy counter source
	raplPaths := []string{
		"/sys/class/powercap/intel-rapl/intel-rapl:0/energy_uj",
		"/sys/class/powercap/intel-rapl:0/energy_uj",
		"/sys/class/powercap/intel-rapl-mmio/intel-rapl-mmio:0/energy_uj",
	}
	for _, p := range raplPaths {
		if f, err := os.Open(p); err == nil {
			s.raplFile = f
			break
		}
	}

	// Initial warm-up sample
	var dummy CPUStats
	_ = s.Sample(&dummy)

	return s, nil
}

// Close releases any held file descriptors.
func (s *CPUSampler) Close() {
	if s.statFile != nil {
		_ = s.statFile.Close()
	}
	if s.freqFile != nil {
		_ = s.freqFile.Close()
	}
	if s.tempFile != nil {
		_ = s.tempFile.Close()
	}
	if s.raplFile != nil {
		_ = s.raplFile.Close()
	}
}

// Sample updates stats in-place with zero heap allocations.
func (s *CPUSampler) Sample(stats *CPUStats) error {
	stats.ModelName = s.modelName

	// 1. CPU Utilization from /proc/stat
	if _, err := s.statFile.Seek(0, io.SeekStart); err == nil {
		n, err := s.statFile.Read(s.buf[:])
		if err == nil && n > 0 {
			tot, idl, pct, pErr := ParseProcStat(s.buf[:n], s.prevTotal, s.prevIdle)
			if pErr == nil {
				s.prevTotal = tot
				s.prevIdle = idl
				stats.UsagePct = pct
			}
		}
	}

	// 2. Scaling Frequency in GHz
	if s.freqFile != nil {
		if _, err := s.freqFile.Seek(0, io.SeekStart); err == nil {
			n, err := s.freqFile.Read(s.smallBuf[:])
			if err == nil && n > 0 {
				khz, _ := ParseUint(s.smallBuf[:n])
				if khz > 0 {
					stats.FreqGHz = float64(khz) / 1000000.0
				}
			}
		}
	}
	if stats.FreqGHz == 0 {
		stats.FreqGHz = readCPUFreqFromInfo()
	}

	// 3. Package Temperature in °C
	if s.tempFile != nil {
		if _, err := s.tempFile.Seek(0, io.SeekStart); err == nil {
			n, err := s.tempFile.Read(s.smallBuf[:])
			if err == nil && n > 0 {
				milliDeg, _ := ParseInt(s.smallBuf[:n])
				if milliDeg > 0 {
					stats.TempC = int(milliDeg / 1000)
				}
			}
		}
	}

	// 4. Power consumption in Watts from RAPL
	now := time.Now()
	if s.raplFile != nil {
		if _, err := s.raplFile.Seek(0, io.SeekStart); err == nil {
			n, err := s.raplFile.Read(s.smallBuf[:])
			if err == nil && n > 0 {
				energyUJ, _ := ParseUint(s.smallBuf[:n])
				if s.prevEnergyUJ > 0 && energyUJ >= s.prevEnergyUJ && !s.prevEnergyTime.IsZero() {
					dtMicro := now.Sub(s.prevEnergyTime).Microseconds()
					if dtMicro > 0 {
						dEnergy := energyUJ - s.prevEnergyUJ
						stats.PowerW = float64(dEnergy) / float64(dtMicro)
					}
				}
				s.prevEnergyUJ = energyUJ
				s.prevEnergyTime = now
			}
		}
	} else {
		stats.PowerW = 0.0
	}

	return nil
}

// readCPUModelName extracts and cleans the CPU model string once at init.
func readCPUModelName() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "CPU"
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[1])
				// Clean verbose boilerplate: (R), (TM), @ freq, etc.
				name = strings.ReplaceAll(name, "(R)", "")
				name = strings.ReplaceAll(name, "(TM)", "")
				name = strings.ReplaceAll(name, "Processor", "")
				if atIdx := strings.Index(name, "@"); atIdx != -1 {
					name = strings.TrimSpace(name[:atIdx])
				}
				name = strings.Join(strings.Fields(name), " ")
				if len(name) > 28 {
					name = name[:28]
				}
				return name
			}
		}
	}
	return "Generic CPU"
}

// readCPUFreqFromInfo falls back to reading MHz from /proc/cpuinfo if sysfs is missing.
func readCPUFreqFromInfo() float64 {
	fd, err := syscall.Open("/proc/cpuinfo", syscall.O_RDONLY, 0)
	if err != nil {
		return 0.0
	}
	defer syscall.Close(fd)

	var buf [2048]byte
	n, err := syscall.Read(fd, buf[:])
	if err != nil || n <= 0 {
		return 0.0
	}

	data := buf[:n]
	tag := []byte("cpu MHz")
	idx := bytes.Index(data, tag)
	if idx >= 0 {
		colonIdx := bytes.IndexByte(data[idx:], ':')
		if colonIdx >= 0 {
			mhzVal, _ := ParseUint(data[idx+colonIdx+1:])
			if mhzVal > 0 {
				return float64(mhzVal) / 1000.0
			}
		}
	}
	return 0.0
}

// findCPUThermalPath searches hwmon and thermal_zone for the CPU package temperature.
func findCPUThermalPath() string {
	// Try /sys/class/hwmon
	matches, _ := filepath.Glob("/sys/class/hwmon/hwmon*")
	for _, m := range matches {
		nameBytes, err := os.ReadFile(filepath.Join(m, "name"))
		if err == nil {
			name := strings.TrimSpace(string(nameBytes))
			if name == "coretemp" || name == "k10temp" || name == "zenpower" || name == "cpu_thermal" {
				// Check temp1_input or package input
				tPath := filepath.Join(m, "temp1_input")
				if _, err := os.Stat(tPath); err == nil {
					return tPath
				}
			}
		}
	}

	// Try /sys/class/thermal
	tzMatches, _ := filepath.Glob("/sys/class/thermal/thermal_zone*")
	for _, tz := range tzMatches {
		typeBytes, err := os.ReadFile(filepath.Join(tz, "type"))
		if err == nil {
			tType := strings.TrimSpace(string(typeBytes))
			if tType == "x86_pkg_temp" || tType == "TCPU" || tType == "cpu-thermal" || strings.Contains(strings.ToLower(tType), "cpu") {
				tPath := filepath.Join(tz, "temp")
				if _, err := os.Stat(tPath); err == nil {
					return tPath
				}
			}
		}
	}

	// Fallback to first available hwmon temp1_input or thermal_zone0/temp
	for _, m := range matches {
		tPath := filepath.Join(m, "temp1_input")
		if _, err := os.Stat(tPath); err == nil {
			return tPath
		}
	}
	for _, tz := range tzMatches {
		tPath := filepath.Join(tz, "temp")
		if _, err := os.Stat(tPath); err == nil {
			return tPath
		}
	}
	return ""
}
