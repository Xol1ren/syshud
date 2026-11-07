package sysinfo

import (
	"testing"
)

// Sample test data replicating /proc/stat
var sampleProcStat = []byte(`cpu  1182245 1191 242209 21415212 23621 54625 18884 0 0 0
cpu0 132483 155 40224 771144 522 3772 948 0 0 0
cpu1 114251 120 11520 823485 1090 1746 1004 0 0 0
intr 38624128 32 0 0 0
ctxt 74163984
btime 1730300000
processes 12345
procs_running 2
procs_blocked 0
softirq 1234567 0 1234 0 0 0 0 0 0 0 0
`)

// Sample test data replicating /proc/meminfo
var sampleProcMeminfo = []byte(`MemTotal:       32258616 kB
MemFree:         2860608 kB
MemAvailable:   21734052 kB
Buffers:            2088 kB
Cached:         20152784 kB
SwapCached:            0 kB
Active:          8408940 kB
Inactive:       17348668 kB
Active(anon):    3579204 kB
Inactive(anon):  3377080 kB
Active(file):    4829736 kB
Inactive(file): 13971588 kB
Unevictable:       18944 kB
Mlocked:           18944 kB
SwapTotal:             0 kB
SwapFree:              0 kB
Dirty:               144 kB
Writeback:             0 kB
AnonPages:       6944620 kB
Mapped:          1069352 kB
Shmem:             27776 kB
KReclaimable:     838528 kB
Slab:            1125272 kB
SReclaimable:     786328 kB
SUnreclaim:       338944 kB
`)

func TestParseProcStat(t *testing.T) {
	tot1, idl1, pct1, err := ParseProcStat(sampleProcStat, 0, 0)
	if err != nil {
		t.Fatalf("ParseProcStat failed: %v", err)
	}
	if tot1 == 0 || idl1 == 0 {
		t.Fatalf("Expected non-zero tot1 and idl1, got tot1=%d, idl1=%d", tot1, idl1)
	}
	if pct1 != 0 {
		t.Fatalf("Expected 0%% utilization on initial sample, got %f", pct1)
	}

	// Simulated next tick with 100 ticks elapsed: 40 active, 60 idle
	sampleNext := []byte(`cpu  1182285 1191 242209 21415272 23621 54625 18884 0 0 0
`)
	tot2, _, pct2, err := ParseProcStat(sampleNext, tot1, idl1)
	if err != nil {
		t.Fatalf("ParseProcStat step 2 failed: %v", err)
	}
	if tot2 <= tot1 {
		t.Fatalf("Expected tot2 > tot1, got tot2=%d, tot1=%d", tot2, tot1)
	}
	if pct2 < 39.0 || pct2 > 41.0 {
		t.Errorf("Expected ~40%% CPU utilization, got %.2f%%", pct2)
	}
}

func TestParseProcMeminfo(t *testing.T) {
	var stats MemStats
	if err := ParseProcMeminfo(sampleProcMeminfo, &stats); err != nil {
		t.Fatalf("ParseProcMeminfo failed: %v", err)
	}

	// 32258616 kB / (1024*1024) = 30.76 GB
	if stats.TotalGB < 30.0 || stats.TotalGB > 32.0 {
		t.Errorf("Unexpected TotalGB: %.2f", stats.TotalGB)
	}
	// Used = Total (32258616) - Available (21734052) = 10524564 kB = 10.03 GB
	if stats.UsedGB < 9.5 || stats.UsedGB > 10.5 {
		t.Errorf("Unexpected UsedGB: %.2f", stats.UsedGB)
	}
	// Cache = Buffers (2088) + Cached (20152784) + SReclaimable (786328) = 20941200 kB = 19.97 GB
	if stats.CacheGB < 19.0 || stats.CacheGB > 21.0 {
		t.Errorf("Unexpected CacheGB: %.2f", stats.CacheGB)
	}
	if stats.UsedPct < 30.0 || stats.UsedPct > 35.0 {
		t.Errorf("Unexpected UsedPct: %.2f%%", stats.UsedPct)
	}
}

func TestDiskSampler(t *testing.T) {
	sampler, err := NewDiskSampler()
	if err != nil {
		t.Fatalf("NewDiskSampler failed: %v", err)
	}
	var stats DiskStats
	if err := sampler.Sample(&stats); err != nil {
		t.Fatalf("Sample disk failed: %v", err)
	}
	if stats.MountPoint != "/home" && stats.MountPoint != "/" {
		t.Errorf("Unexpected mount point: %s", stats.MountPoint)
	}
	if stats.TotalTB <= 0 {
		t.Errorf("Expected positive TotalTB, got %.4f", stats.TotalTB)
	}
}

func TestGPUSamplerGraceful(t *testing.T) {
	sampler := NewGPUSampler()
	defer sampler.Close()

	var stats GPUStats
	if err := sampler.Sample(&stats); err != nil {
		t.Fatalf("Sample GPU failed: %v", err)
	}
	if !stats.Available {
		if stats.ModelName != "N/A" {
			t.Errorf("Expected N/A for unavailable GPU, got %s", stats.ModelName)
		}
	} else {
		if stats.ModelName == "" {
			t.Errorf("Expected non-empty model name for available GPU")
		}
	}
}

func BenchmarkParseProcStat(b *testing.B) {
	b.ReportAllocs()
	var prevTotal, prevIdle uint64
	for i := 0; i < b.N; i++ {
		tot, idl, _, _ := ParseProcStat(sampleProcStat, prevTotal, prevIdle)
		prevTotal = tot
		prevIdle = idl
	}
}

func BenchmarkParseProcMeminfo(b *testing.B) {
	b.ReportAllocs()
	var stats MemStats
	for i := 0; i < b.N; i++ {
		_ = ParseProcMeminfo(sampleProcMeminfo, &stats)
	}
}

func BenchmarkDiskSample(b *testing.B) {
	sampler, err := NewDiskSampler()
	if err != nil {
		b.Fatalf("NewDiskSampler failed: %v", err)
	}
	defer sampler.Close()

	var stats DiskStats
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = sampler.Sample(&stats)
	}
}

func BenchmarkCollectorSampleLive(b *testing.B) {
	collector, err := NewCollector()
	if err != nil {
		b.Skipf("Collector init skipped: %v", err)
	}
	defer collector.Close()

	var stats SystemStats
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = collector.Sample(&stats)
	}
}
