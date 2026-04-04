// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// procstats.go — Linux /proc helpers for OS-level memory and CPU measurement.
// These are best-effort: all functions return zero / empty struct on any error
// so callers never need to handle errors from snapshot-taking code paths.

package benchmark

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// --------------------------------------------------------------------------
// Peak RSS
// --------------------------------------------------------------------------

// readPeakRSSKiB returns the peak resident-set-size of this process in KiB by
// reading the VmHWM field from /proc/self/status.  Returns 0 on any error.
func readPeakRSSKiB() int64 {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return 0
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "VmHWM:") {
			continue
		}
		// Format: "VmHWM:    1234 kB"
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0
		}
		v, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return 0
		}
		return v
	}
	return 0
}

// --------------------------------------------------------------------------
// Per-process CPU
// --------------------------------------------------------------------------

// cpuStats holds raw kernel CPU-tick values from /proc/self/stat.
type cpuStats struct {
	userTicks uint64
	sysTicks  uint64
}

// ticksPerSec is the Linux CLK_TCK constant. On almost every Linux platform
// this is 100 Hz (each tick = 10 ms).  If it ever differs, adjust here.
const ticksPerSec = 100

// ticksToMs converts kernel clock ticks to milliseconds using ticksPerSec.
func ticksToMs(ticks uint64) int64 {
	return int64(ticks) * 1000 / ticksPerSec
}

// readProcCPU reads utime and stime (in kernel clock ticks) from
// /proc/self/stat and returns them as a cpuStats value.
// Returns an error only on file I/O or parse failures.
func readProcCPU() (cpuStats, error) {
	data, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return cpuStats{}, fmt.Errorf("procstats: reading /proc/self/stat: %w", err)
	}

	// /proc/<pid>/stat format (man 5 proc):
	//   (1)pid (2)comm (3)state (4)ppid ... (14)utime (15)stime ...
	//
	// The comm field (2) may contain spaces and parentheses, so we locate the
	// last ')' and parse fields relative to that position.
	line := string(data)
	rp := strings.LastIndex(line, ")")
	if rp < 0 {
		return cpuStats{}, fmt.Errorf("procstats: malformed /proc/self/stat (no closing paren)")
	}

	// After ") " we have: state ppid pgrp session tty_nr tpgid flags
	//                      minflt cminflt majflt cmajflt utime stime ...
	// That is 12 whitespace-separated tokens before utime (index 11) and
	// stime (index 12) when zero-indexed from the character after ')'.
	rest := strings.TrimSpace(line[rp+1:])
	fields := strings.Fields(rest)
	if len(fields) < 13 {
		return cpuStats{}, fmt.Errorf("procstats: too few fields after comm in /proc/self/stat (got %d)", len(fields))
	}

	utime, err := strconv.ParseUint(fields[11], 10, 64)
	if err != nil {
		return cpuStats{}, fmt.Errorf("procstats: parsing utime field: %w", err)
	}
	stime, err := strconv.ParseUint(fields[12], 10, 64)
	if err != nil {
		return cpuStats{}, fmt.Errorf("procstats: parsing stime field: %w", err)
	}
	return cpuStats{userTicks: utime, sysTicks: stime}, nil
}

// --------------------------------------------------------------------------
// System-wide CPU
// --------------------------------------------------------------------------

// systemCPUInfo holds the raw aggregated CPU jiffies from /proc/stat.
type systemCPUInfo struct {
	user    uint64
	nice    uint64
	system  uint64
	idle    uint64
	iowait  uint64
	irq     uint64
	softirq uint64
	steal   uint64
}

// total returns the sum of all CPU-tick categories.
func (s systemCPUInfo) total() uint64 {
	return s.user + s.nice + s.system + s.idle + s.iowait + s.irq + s.softirq + s.steal
}

// idleTotal returns idle + iowait ticks (both represent "not active" CPU time).
func (s systemCPUInfo) idleTotal() uint64 {
	return s.idle + s.iowait
}

// readSystemCPU reads the first aggregated "cpu" line from /proc/stat.
// Returns an error on I/O or parse failure.
func readSystemCPU() (systemCPUInfo, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return systemCPUInfo{}, fmt.Errorf("procstats: opening /proc/stat: %w", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		// Format: cpu user nice system idle iowait irq softirq steal [guest guest_nice]
		fields := strings.Fields(line)
		if len(fields) < 5 { // need at least cpu user nice system idle
			return systemCPUInfo{}, fmt.Errorf("procstats: too few fields in /proc/stat cpu line (%d)", len(fields))
		}
		parse := func(idx int) uint64 {
			if idx >= len(fields) {
				return 0
			}
			v, _ := strconv.ParseUint(fields[idx], 10, 64)
			return v
		}
		return systemCPUInfo{
			user:    parse(1),
			nice:    parse(2),
			system:  parse(3),
			idle:    parse(4),
			iowait:  parse(5),
			irq:     parse(6),
			softirq: parse(7),
			steal:   parse(8),
		}, nil
	}
	if err := sc.Err(); err != nil {
		return systemCPUInfo{}, fmt.Errorf("procstats: scanning /proc/stat: %w", err)
	}
	return systemCPUInfo{}, fmt.Errorf("procstats: no 'cpu' line found in /proc/stat")
}

// systemCPUPercent computes the system-wide CPU utilisation percentage between
// two /proc/stat snapshots.  The result is a percentage of ONE CPU core and
// may exceed 100 on multi-core machines.  Returns 0 when no ticks have
// advanced (e.g. snapshots taken too close together).
func systemCPUPercent(before, after systemCPUInfo) float64 {
	totalDelta := float64(after.total()) - float64(before.total())
	if totalDelta <= 0 {
		return 0
	}
	idleDelta := float64(after.idleTotal()) - float64(before.idleTotal())
	return (1 - idleDelta/totalDelta) * 100
}

// --------------------------------------------------------------------------
// System memory (/proc/meminfo)
// --------------------------------------------------------------------------

// sysMemInfo holds system-wide memory counters from /proc/meminfo (all in KiB).
type sysMemInfo struct {
	cachedKiB    int64 // Linux page cache (file-backed pages)
	anonPagesKiB int64 // anonymous mapped pages (heap, stacks, socket buffers)
}

// readSysMemInfo reads Cached and AnonPages from /proc/meminfo.
// Returns a zero struct on any error.
func readSysMemInfo() sysMemInfo {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return sysMemInfo{}
	}
	defer f.Close()

	var info sysMemInfo
	found := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() && found < 2 {
		line := sc.Text()
		var dest *int64
		switch {
		case strings.HasPrefix(line, "Cached:"):
			dest = &info.cachedKiB
		case strings.HasPrefix(line, "AnonPages:"):
			dest = &info.anonPagesKiB
		default:
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			v, _ := strconv.ParseInt(fields[1], 10, 64)
			*dest = v
			found++
		}
	}
	return info
}

// readCurrentRSSKiB returns the current resident-set-size of this process in
// KiB by reading VmRSS from /proc/self/status. Returns 0 on any error.
func readCurrentRSSKiB() int64 {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return 0
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "VmRSS:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0
		}
		v, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return 0
		}
		return v
	}
	return 0
}

// --------------------------------------------------------------------------
// Disk page I/O (/proc/vmstat)
// --------------------------------------------------------------------------

// vmStat holds cumulative page-in/out counters from /proc/vmstat.
type vmStat struct {
	pgpgin  uint64 // pages read from disk (cumulative)
	pgpgout uint64 // pages written to disk (cumulative)
}

// readVMStat reads pgpgin and pgpgout from /proc/vmstat.
// Non-zero delta between two snapshots means data was read from (or written
// to) disk during that interval, ruling out pure-network benchmarks.
// Returns a zero struct on any error.
func readVMStat() vmStat {
	f, err := os.Open("/proc/vmstat")
	if err != nil {
		return vmStat{}
	}
	defer f.Close()

	var stat vmStat
	found := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() && found < 2 {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		v, _ := strconv.ParseUint(fields[1], 10, 64)
		switch fields[0] {
		case "pgpgin":
			stat.pgpgin = v
			found++
		case "pgpgout":
			stat.pgpgout = v
			found++
		}
	}
	return stat
}
