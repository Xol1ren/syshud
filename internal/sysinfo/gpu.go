package sysinfo

import (
	"bytes"
	"strings"

	"github.com/ebitengine/purego"
)

type nvmlReturn int

const nvmlSuccess nvmlReturn = 0

type nvmlMemory struct {
	Total uint64
	Free  uint64
	Used  uint64
}

type nvmlUtilization struct {
	GPU    uint32
	Memory uint32
}

// GPUSampler queries NVIDIA hardware metrics via dynamic loading of libnvidia-ml.so.
type GPUSampler struct {
	libHandle uintptr
	device    uintptr
	available bool
	modelName string

	// Function pointers registered via purego
	nvmlInit                      func() nvmlReturn
	nvmlShutdown                  func() nvmlReturn
	nvmlDeviceGetCount            func(*uint32) nvmlReturn
	nvmlDeviceGetHandleByIndex    func(uint32, *uintptr) nvmlReturn
	nvmlDeviceGetName             func(uintptr, *byte, uint32) nvmlReturn
	nvmlDeviceGetUtilizationRates func(uintptr, *nvmlUtilization) nvmlReturn
	nvmlDeviceGetMemoryInfo       func(uintptr, *nvmlMemory) nvmlReturn
	nvmlDeviceGetTemperature      func(uintptr, uint32, *uint32) nvmlReturn
	nvmlDeviceGetPowerUsage       func(uintptr, *uint32) nvmlReturn

	// Pre-allocated stack-like structs to avoid heap escape
	mem  nvmlMemory
	util nvmlUtilization
}

// NewGPUSampler attempts to dynamically load libnvidia-ml.so.
// If the library or GPU is absent, it gracefully falls back without failing.
func NewGPUSampler() *GPUSampler {
	s := &GPUSampler{
		available: false,
		modelName: "N/A",
	}

	// Try standard library names
	libPaths := []string{
		"libnvidia-ml.so.1",
		"libnvidia-ml.so",
		"/usr/lib64/libnvidia-ml.so.1",
		"/usr/lib/x86_64-linux-gnu/libnvidia-ml.so.1",
		"/lib64/libnvidia-ml.so.1",
	}

	var lib uintptr
	var err error
	for _, p := range libPaths {
		lib, err = purego.Dlopen(p, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err == nil && lib != 0 {
			break
		}
	}
	if lib == 0 {
		return s
	}
	s.libHandle = lib

	// Register symbols
	registerFunc := func(target any, names ...string) bool {
		for _, name := range names {
			purego.RegisterLibFunc(target, lib, name)
			// purego sets the func pointer if found
		}
		return true
	}

	registerFunc(&s.nvmlInit, "nvmlInit_v2", "nvmlInit")
	registerFunc(&s.nvmlShutdown, "nvmlShutdown")
	registerFunc(&s.nvmlDeviceGetCount, "nvmlDeviceGetCount_v2", "nvmlDeviceGetCount")
	registerFunc(&s.nvmlDeviceGetHandleByIndex, "nvmlDeviceGetHandleByIndex_v2", "nvmlDeviceGetHandleByIndex")
	registerFunc(&s.nvmlDeviceGetName, "nvmlDeviceGetName")
	registerFunc(&s.nvmlDeviceGetUtilizationRates, "nvmlDeviceGetUtilizationRates")
	registerFunc(&s.nvmlDeviceGetMemoryInfo, "nvmlDeviceGetMemoryInfo")
	registerFunc(&s.nvmlDeviceGetTemperature, "nvmlDeviceGetTemperature")
	registerFunc(&s.nvmlDeviceGetPowerUsage, "nvmlDeviceGetPowerUsage")

	if s.nvmlInit == nil {
		return s
	}

	if ret := s.nvmlInit(); ret != nvmlSuccess {
		return s
	}

	if s.nvmlDeviceGetCount == nil {
		return s
	}
	var count uint32
	if ret := s.nvmlDeviceGetCount(&count); ret != nvmlSuccess || count == 0 {
		return s
	}

	if s.nvmlDeviceGetHandleByIndex == nil {
		return s
	}
	if ret := s.nvmlDeviceGetHandleByIndex(0, &s.device); ret != nvmlSuccess || s.device == 0 {
		return s
	}

	// Read and clean GPU Model Name
	if s.nvmlDeviceGetName != nil {
		var nameBuf [96]byte
		if ret := s.nvmlDeviceGetName(s.device, &nameBuf[0], 96); ret == nvmlSuccess {
			raw := string(bytes.TrimRight(nameBuf[:], "\x00 \t\n\r"))
			// Clean up verbose branding if needed
			clean := strings.TrimSpace(raw)
			clean = strings.ReplaceAll(clean, "NVIDIA ", "")
			clean = strings.ReplaceAll(clean, "Laptop GPU", "")
			clean = strings.TrimSpace(clean)
			s.modelName = clean
		}
	}
	if s.modelName == "" || s.modelName == "N/A" {
		s.modelName = "NVIDIA GPU"
	}

	s.available = true
	return s
}

// Close gracefully shuts down the NVML library session if open.
func (s *GPUSampler) Close() {
	if s.available && s.nvmlShutdown != nil {
		_ = s.nvmlShutdown()
		s.available = false
	}
}

// Sample queries GPU metrics in-place with zero heap allocations.
func (s *GPUSampler) Sample(stats *GPUStats) error {
	if !s.available {
		stats.Available = false
		stats.ModelName = "N/A"
		stats.UsagePct = 0
		stats.VRAMUsedGB = 0
		stats.VRAMTotalGB = 0
		stats.VRAMFreeGB = 0
		stats.VRAMUsedPct = 0
		stats.TempC = 0
		stats.PowerW = 0
		return nil
	}

	stats.Available = true
	stats.ModelName = s.modelName

	// 1. GPU Core Utilization %
	if s.nvmlDeviceGetUtilizationRates != nil {
		if ret := s.nvmlDeviceGetUtilizationRates(s.device, &s.util); ret == nvmlSuccess {
			stats.UsagePct = int(s.util.GPU)
			if stats.UsagePct > 100 {
				stats.UsagePct = 100
			}
		}
	}

	// 2. VRAM Memory
	if s.nvmlDeviceGetMemoryInfo != nil {
		if ret := s.nvmlDeviceGetMemoryInfo(s.device, &s.mem); ret == nvmlSuccess {
			stats.VRAMTotalGB = float64(s.mem.Total) / bytesPerGB
			stats.VRAMUsedGB = float64(s.mem.Used) / bytesPerGB
			stats.VRAMFreeGB = float64(s.mem.Free) / bytesPerGB
			if s.mem.Total > 0 {
				stats.VRAMUsedPct = float64(s.mem.Used) / float64(s.mem.Total) * 100.0
			}
		}
	}

	// 3. Core Temperature °C (0 = NVML_TEMPERATURE_GPU)
	if s.nvmlDeviceGetTemperature != nil {
		var temp uint32
		if ret := s.nvmlDeviceGetTemperature(s.device, 0, &temp); ret == nvmlSuccess {
			stats.TempC = int(temp)
		}
	}

	// 4. Instant Power Draw in Watts (milliwatts / 1000.0)
	if s.nvmlDeviceGetPowerUsage != nil {
		var powerMW uint32
		if ret := s.nvmlDeviceGetPowerUsage(s.device, &powerMW); ret == nvmlSuccess {
			stats.PowerW = float64(powerMW) / 1000.0
		}
	}

	return nil
}
