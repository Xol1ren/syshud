package sysinfo

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

var (
	tagMemTotal     = []byte("MemTotal:")
	tagMemFree      = []byte("MemFree:")
	tagMemAvailable = []byte("MemAvailable:")
	tagBuffers      = []byte("Buffers:")
	tagCached       = []byte("Cached:")
	tagSReclaimable = []byte("SReclaimable:")
)

const bytesPerGB = 1024.0 * 1024.0 * 1024.0

// ParseProcMeminfo parses memory statistics from /proc/meminfo buffer content.
// Values in /proc/meminfo are in kB (1024 bytes).
// Computes:
//   - Used = Total - Available
//   - Cache = Buffers + Cached + SReclaimable
//   - Completely omits Swap.
//
// Guarantees zero heap allocations.
func ParseProcMeminfo(data []byte, stats *MemStats) error {
	var totalKB, freeKB, availKB, buffersKB, cachedKB, sreclaimableKB uint64
	d := data
	for len(d) > 0 {
		nl := bytes.IndexByte(d, '\n')
		var line []byte
		if nl >= 0 {
			line = d[:nl]
			d = d[nl+1:]
		} else {
			line = d
			d = nil
		}
		if bytes.HasPrefix(line, tagMemTotal) {
			totalKB, _ = ParseUint(line[len(tagMemTotal):])
		} else if bytes.HasPrefix(line, tagMemAvailable) {
			availKB, _ = ParseUint(line[len(tagMemAvailable):])
		} else if bytes.HasPrefix(line, tagMemFree) {
			freeKB, _ = ParseUint(line[len(tagMemFree):])
		} else if bytes.HasPrefix(line, tagBuffers) {
			buffersKB, _ = ParseUint(line[len(tagBuffers):])
		} else if bytes.HasPrefix(line, tagCached) {
			cachedKB, _ = ParseUint(line[len(tagCached):])
		} else if bytes.HasPrefix(line, tagSReclaimable) {
			sreclaimableKB, _ = ParseUint(line[len(tagSReclaimable):])
		}
	}

	totalBytes := totalKB * 1024
	var usedBytes uint64
	if availKB > 0 {
		if totalKB >= availKB {
			usedBytes = (totalKB - availKB) * 1024
		}
	} else {
		// Fallback for kernels older than 3.14 lacking MemAvailable
		nonAvail := freeKB + buffersKB + cachedKB + sreclaimableKB
		if totalKB >= nonAvail {
			usedBytes = (totalKB - nonAvail) * 1024
		}
	}

	cacheBytes := (buffersKB + cachedKB + sreclaimableKB) * 1024
	freeBytes := freeKB * 1024
	availBytes := availKB * 1024

	stats.TotalGB = float64(totalBytes) / bytesPerGB
	stats.UsedGB = float64(usedBytes) / bytesPerGB
	stats.AvailableGB = float64(availBytes) / bytesPerGB
	stats.FreeGB = float64(freeBytes) / bytesPerGB
	stats.CacheGB = float64(cacheBytes) / bytesPerGB

	if totalBytes > 0 {
		stats.UsedPct = float64(usedBytes) / float64(totalBytes) * 100.0
		if stats.UsedPct > 100.0 {
			stats.UsedPct = 100.0
		}
	}

	return nil
}

// MemSampler samples memory metrics from /proc/meminfo with zero allocations.
type MemSampler struct {
	file *os.File
	buf  [2048]byte
}

// NewMemSampler creates a MemSampler holding an open /proc/meminfo descriptor.
func NewMemSampler() (*MemSampler, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/meminfo: %w", err)
	}
	return &MemSampler{file: f}, nil
}

// Close releases the open file descriptor.
func (s *MemSampler) Close() {
	if s.file != nil {
		_ = s.file.Close()
	}
}

// Sample updates stats in-place with zero heap allocations.
func (s *MemSampler) Sample(stats *MemStats) error {
	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	n, err := s.file.Read(s.buf[:])
	if err != nil && err != io.EOF {
		return err
	}
	return ParseProcMeminfo(s.buf[:n], stats)
}

