package sysinfo

// CPUStats holds instantaneous CPU telemetry metrics.
type CPUStats struct {
	ModelName string
	UsagePct  float64
	TempC     int
	FreqGHz   float64
	PowerW    float64
}

// MemStats holds memory telemetry metrics in Gigabytes (GB).
type MemStats struct {
	TotalGB     float64
	UsedGB      float64
	AvailableGB float64
	FreeGB      float64
	CacheGB     float64
	UsedPct     float64
}

// DiskStats holds filesystem capacity metrics in Terabytes (TB).
type DiskStats struct {
	MountPoint string
	TotalTB    float64
	UsedTB     float64
	FreeTB     float64
	UsedPct    float64
}

// GPUStats holds NVIDIA GPU hardware metrics.
type GPUStats struct {
	Available   bool
	ModelName   string
	UsagePct    int
	VRAMUsedGB  float64
	VRAMTotalGB float64
	VRAMFreeGB  float64
	VRAMUsedPct float64
	TempC       int
	PowerW      float64
}

// SystemStats is the aggregate container for all hardware telemetry.
type SystemStats struct {
	CPU  CPUStats
	Mem  MemStats
	Disk DiskStats
	GPU  GPUStats
}
