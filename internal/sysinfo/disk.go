package sysinfo

import (
	"os"
	"syscall"
	"unsafe"
)

const bytesPerTB = 1024.0 * 1024.0 * 1024.0 * 1024.0

// DiskSampler samples filesystem capacity for /home (fallback to /).
type DiskSampler struct {
	mountPoint      string
	mountPointBytes [64]byte
	stat            syscall.Statfs_t
}

// NewDiskSampler checks if /home exists; if not, falls back to /.
func NewDiskSampler() (*DiskSampler, error) {
	target := "/home"
	if _, err := os.Stat("/home"); err != nil {
		target = "/"
	}
	s := &DiskSampler{
		mountPoint: target,
	}
	copy(s.mountPointBytes[:], target)
	s.mountPointBytes[len(target)] = 0 // null-terminated C string for syscall
	return s, nil
}

// Close is a no-op for DiskSampler.
func (s *DiskSampler) Close() {}

// Sample updates stats in-place via direct statfs syscall with zero heap allocations.
func (s *DiskSampler) Sample(stats *DiskStats) error {
	stats.MountPoint = s.mountPoint

	r1, _, errno := syscall.Syscall(
		syscall.SYS_STATFS,
		uintptr(unsafe.Pointer(&s.mountPointBytes[0])),
		uintptr(unsafe.Pointer(&s.stat)),
		0,
	)
	if int(r1) < 0 {
		return errno
	}

	bsize := uint64(s.stat.Bsize)
	totalBytes := s.stat.Blocks * bsize
	// Bavail is the number of free blocks available to unprivileged users
	freeBytes := s.stat.Bavail * bsize

	var usedBytes uint64
	if totalBytes >= freeBytes {
		usedBytes = totalBytes - freeBytes
	}

	stats.TotalTB = float64(totalBytes) / bytesPerTB
	stats.FreeTB = float64(freeBytes) / bytesPerTB
	stats.UsedTB = float64(usedBytes) / bytesPerTB

	if totalBytes > 0 {
		stats.UsedPct = float64(usedBytes) / float64(totalBytes) * 100.0
		if stats.UsedPct > 100.0 {
			stats.UsedPct = 100.0
		}
	} else {
		stats.UsedPct = 0.0
	}

	return nil
}
