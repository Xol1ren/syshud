package sysinfo

// Collector orchestrates all hardware and kernel telemetry samplers.
type Collector struct {
	cpu  *CPUSampler
	mem  *MemSampler
	disk *DiskSampler
	gpu  *GPUSampler
}

// NewCollector initializes each hardware telemetry engine.
func NewCollector() (*Collector, error) {
	cpu, err := NewCPUSampler()
	if err != nil {
		return nil, err
	}

	mem, err := NewMemSampler()
	if err != nil {
		cpu.Close()
		return nil, err
	}

	disk, err := NewDiskSampler()
	if err != nil {
		cpu.Close()
		mem.Close()
		return nil, err
	}

	gpu := NewGPUSampler()

	return &Collector{
		cpu:  cpu,
		mem:  mem,
		disk: disk,
		gpu:  gpu,
	}, nil
}

// Close gracefully releases all held kernel file descriptors and library handles.
func (c *Collector) Close() {
	if c.cpu != nil {
		c.cpu.Close()
	}
	if c.mem != nil {
		c.mem.Close()
	}
	if c.disk != nil {
		c.disk.Close()
	}
	if c.gpu != nil {
		c.gpu.Close()
	}
}

// Sample refreshes all hardware metrics in-place with zero heap allocations.
func (c *Collector) Sample(stats *SystemStats) error {
	if err := c.cpu.Sample(&stats.CPU); err != nil {
		return err
	}
	if err := c.mem.Sample(&stats.Mem); err != nil {
		return err
	}
	if err := c.disk.Sample(&stats.Disk); err != nil {
		return err
	}
	if err := c.gpu.Sample(&stats.GPU); err != nil {
		return err
	}
	return nil
}

