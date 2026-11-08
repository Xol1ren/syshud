# syshud

[![CI Pipeline](https://github.com/Xol1ren/syshud/actions/workflows/ci.yml/badge.svg)](https://github.com/Xol1ren/syshud/actions/workflows/ci.yml)

A compact terminal HUD dashboard in Go with a lightweight footprint (< 3 MB RSS) and zero heap allocations per telemetry cycle.

```text
┌─ CPU [Intel Core Ultra 9 275HX] ─────┐┌─ MEMORY ─────────────────────────────┐
│ Usage: [████████░░░░░░░░] 38%        ││ RAM:   12.4 / 31.8 GB (39%)          │
│ Temp:  54°C    Freq: 3.2 GHz  Power: 28W││ Cache: 6.8 GB | Free: 12.6 GB        │
└──────────────────────────────────────┘└──────────────────────────────────────┘
┌─ GPU [GeForce RTX 5060] ─────────────┐┌─ DISK [/home] ───────────────────────┐
│ Usage: [████░░░░░░░░░░░░] 22%        ││ Used:  0.42 TB / 1.02 TB (41%)       │
│ VRAM:  2.1 / 8.0 GB (26%)            ││ Free:  0.60 TB                       │
│ Temp:  49°C    Power: 42W            ││ Bar:   [████████░░░░░░░░] 41%        │
└──────────────────────────────────────┘└──────────────────────────────────────┘
 syshud • tick: 1000ms • [q] quit
```

---

## Core Principles & Design

1. **Compact Single-Binary HUD**:
   - Takes exactly **10–12 terminal rows** total.
   - Symmetrically balanced 2x2 grid that automatically scales across 2 columns on window resize (`tea.WindowSizeMsg`).

2. **Zero-Allocation Runtime Telemetry Engine**:
   - Scrapes Linux `/proc` and `/sys` directly through pre-opened file descriptors and stack buffers (`[2048]byte`).
   - Zero heap allocations during parsing and refresh ticks (`0 B/op`, `0 allocs/op`).
   - Absolutely **NO `gopsutil`**, **NO heavy dependencies**.

3. **Minimalist 4-Card Grid**:
   - **CPU**: Aggregated utilization %, core scaling frequency (GHz), package temperature (°C), and Intel/AMD RAPL energy counter power sampling (Watts, with graceful fallback to 0W).
   - **MEMORY**: Real RAM used (`Total - Available`), cache (`Buffers + Cached + SReclaimable`), and free memory in Gigabytes (GB). Completely omits Swap.
   - **GPU**: Pure Go dynamic library loading of `libnvidia-ml.so` via `purego`. No CGO, no NVML headers required at compile time. Queries core utilization (%), VRAM Used/Total/Free (GB), core temperature (°C), and instant power draw (Watts). Gracefully reports `N/A` without crashing if NVIDIA drivers are absent.
   - **DISK**: Targets strictly the `/home` mount point (fallback to `/` only if `/home` does not exist). Uses direct `syscall.Statfs` and formats capacity strictly in Terabytes (TB) with two decimal places (`Used: 0.42 TB / 1.02 TB (41%) | Free: 0.60 TB`). Displays the usage progress bar with no generic status labels.

---

## Telemetry Architecture (`internal/sysinfo`)

| Subsystem | Linux Kernel Source | Extraction Mechanism | Zero-Alloc Strategy |
| :--- | :--- | :--- | :--- |
| **CPU Usage** | `/proc/stat` | Delta computation between ticks ($\Delta\text{Total}$, $\Delta\text{Idle}$) | In-place ASCII parsing of byte buffers |
| **CPU Frequency** | `/sys/devices/system/cpu/cpu*/cpufreq/scaling_cur_freq` | Scaling frequency in kHz converted to GHz (fallback to `/proc/cpuinfo`) | Fixed stack buffer read |
| **CPU Temp** | `/sys/class/hwmon/` or `/sys/class/thermal/` | Auto-detects `coretemp`, `k10temp`, `zenpower`, or `x86_pkg_temp` in millidegrees | Re-seekable descriptor |
| **CPU Power** | `/sys/class/powercap/intel-rapl/intel-rapl:0/energy_uj` | $\Delta\text{energy\_uj} / \Delta\text{time\_us}$ | Auto-fallback to 0W on restricted perms |
| **Memory** | `/proc/meminfo` | `Total - Available`, `Buffers + Cached + SReclaimable` | Reusable 2KB buffer, tag scanning |
| **Disk** | `/home` (or `/`) | Direct `syscall.SYS_STATFS` assembly syscall | Stack-allocated `Statfs_t`, pre-built null-terminated path |
| **GPU** | `libnvidia-ml.so.1` / `libnvidia-ml.so` | Runtime `purego.Dlopen` and dynamic symbol binding | Graceful fallback if driver missing |

---

## Performance & Memory Footprint

### Benchmark Results (`go test -bench=. -benchmem ./...`)

```text
goos: linux
goarch: amd64
pkg: syshud/internal/sysinfo
cpu: Intel(R) Core(TM) Ultra 9 275HX
BenchmarkParseProcStat-24          30643142      38.48 ns/op       0 B/op     0 allocs/op
BenchmarkParseProcMeminfo-24        3677278     334.00 ns/op       0 B/op     0 allocs/op
BenchmarkDiskSample-24              3393081     328.00 ns/op       0 B/op     0 allocs/op
PASS
```

- **Runtime Memory (RSS)**: Baseline < 3 MB RSS.
- **CPU Overhead**: < 0.05% CPU utilization at 1000ms tick interval.
- **Binary Size**: ~3.2 MB stripped (`-ldflags="-s -w"`).

---

## Build & Installation

### Prerequisites
- Linux kernel $\ge 4.4$ (x86_64 or aarch64)
- Go $\ge 1.25.11$ (as required in `go.mod`)
- `make`

### Building

```bash
# Compile optimized binary to bin/syshud
make build

# Run unit tests and allocation benchmarks
make test

# Launch HUD directly
make run
```

---

## Limitations

- **Linux only:** Relies on direct Linux `/proc` and `/sys` filesystem interfaces (`/proc/stat`, `/proc/meminfo`, `/sys/class/hwmon`) and Linux-specific syscalls (`syscall.SYS_STATFS`). Does not support macOS or Windows.
- **NVIDIA GPU only via NVML:** GPU metrics require an NVIDIA GPU with `libnvidia-ml.so.1` or `libnvidia-ml.so` installed. AMD and Intel integrated/discrete GPUs are not supported; the GPU panel displays `N/A` if NVIDIA libraries or hardware are absent.
- **Elevated permissions required for CPU power (RAPL):** Linux kernels $\ge 5.10$ restrict `/sys/class/powercap/intel-rapl/*/energy_uj` to `root` (mode `0400`). Without root privileges or custom udev rules, CPU power draw reports `0W`.
- **Fixed disk target (`/home`):** Disk telemetry inspects strictly the `/home` mount point (falling back to `/` only if `/home` does not exist). Arbitrary mount points or multi-disk monitoring are not supported.
- **Single-GPU monitoring:** The NVML sampler attaches to the first GPU device (`index 0`). Multi-GPU setups are not aggregated.

---

## RAPL Power & Sudoers Guide

On modern Linux kernels ($\ge 5.10$), access to Intel/AMD RAPL hardware energy counters (`/sys/class/powercap/intel-rapl/*/energy_uj`) is restricted to `root` (`0400` mode) to mitigate the PLATYPUS side-channel vulnerability (CVE-2020-8694).

`syshud` gracefully handles this by falling back to `0W` when run unprivileged. If you want live package power wattage telemetry, configure one of the following:

### Option 1: Sudoers Entry (Fastest)

Allow running `syshud` as root without entering a password:

```sudoers
# Add to /etc/sudoers.d/syshud (via sudo visudo -f /etc/sudoers.d/syshud)
%wheel ALL=(root) NOPASSWD: /usr/local/bin/syshud
# Or for Debian/Ubuntu:
%sudo ALL=(root) NOPASSWD: /usr/local/bin/syshud
```

Then run:
```bash
sudo syshud
```

### Option 2: Persistent udev Rule (Recommended for Workstations)

Grant read access to unprivileged users automatically on boot:

```bash
echo 'SUBSYSTEM=="powercap", KERNEL=="intel-rapl:*", ACTION=="add", RUN+="/bin/chmod 0444 /sys/class/powercap/%k/energy_uj"' | sudo tee /etc/udev/rules.d/99-intel-rapl.rules
sudo udevadm control --reload && sudo udevadm trigger
```

---

## Keyboard Controls

| Key | Action |
| :--- | :--- |
| `q` or `Q` | Clean quit and restore terminal screen |
| `Esc` | Exit HUD |
| `Ctrl+C` | Force exit |

---

## Project Structure

```text
.
├── cmd/
│   └── hud/
│       └── main.go           # Application entrypoint & Bubble Tea runner
├── internal/
│   ├── sysinfo/
│   │   ├── collector.go      # Unified telemetry manager
│   │   ├── cpu.go            # /proc/stat, cpufreq, thermal, RAPL engine
│   │   ├── disk.go           # /home Statfs zero-alloc engine
│   │   ├── gpu.go            # Pure Go NVML dynamic loader
│   │   ├── memory.go         # /proc/meminfo zero-alloc parser
│   │   ├── parse.go          # Fast ASCII integer conversion primitives
│   │   ├── types.go          # Hardware metric structs
│   │   └── sysinfo_test.go   # Zero-alloc benchmarks & unit tests
│   └── ui/
│       ├── bar.go            # Custom progress bar renderer
│       ├── card.go           # Bordered 2-column box renderer
│       ├── hud.go            # Bubble Tea model (Update, View, Layout)
│       ├── styles.go         # Lipgloss color themes & styles
│       └── ui_test.go        # Terminal layout and sizing tests
├── Makefile                  # Build automation (build, test, run)
├── README.md                 # Documentation & schema
├── go.mod
└── go.sum
```

