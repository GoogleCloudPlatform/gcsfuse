package fsutil

import (
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workloadprofiler"
	"sync"
)

type DataAccessStats struct {
	SequentialReadCount     int64
	RandomReadCount         int64
	TotalAccessedFileHandle int64
	TotalAccessedInode      int64
	TotalSizeReadAccessed   int64
}

type FileSystemProfilerSource struct {
	workloadprofiler.ProfilerSource

	stats map[string]DataAccessStats
	mu    sync.RWMutex
}

func NewFileSystemProfilerSource() *FileSystemProfilerSource {
	fsps := &FileSystemProfilerSource{
		stats: make(map[string]DataAccessStats),
	}

	workloadprofiler.AddProfilerSource(fsps) // Register this source
	return fsps
}

func (fsps *FileSystemProfilerSource) IncrementSequentialReadCount(op string) {
	fsps.mu.Lock()
	defer fsps.mu.Unlock()
	stats := fsps.stats[op]
	stats.SequentialReadCount++
	fsps.stats[op] = stats
}

func (fsps *FileSystemProfilerSource) IncrementRandomReadCount(op string) {
	fsps.mu.Lock()
	defer fsps.mu.Unlock()
	stats := fsps.stats[op]
	stats.RandomReadCount++
	fsps.stats[op] = stats
}

func (fsps *FileSystemProfilerSource) IncrementTotalAccessedFileHandle(op string) {
	fsps.mu.Lock()
	defer fsps.mu.Unlock()
	stats := fsps.stats[op]
	stats.TotalAccessedFileHandle++
	fsps.stats[op] = stats
}

func (fsps *FileSystemProfilerSource) IncrementTotalAccessedInode(op string) {
	fsps.mu.Lock()
	defer fsps.mu.Unlock()
	stats := fsps.stats[op]
	stats.TotalAccessedInode++
	fsps.stats[op] = stats
}

func (fsps *FileSystemProfilerSource) IncrementTotalSizeReadAccessed(op string, size int64) {
	fsps.mu.Lock()
	defer fsps.mu.Unlock()
	stats := fsps.stats[op]
	stats.TotalSizeReadAccessed += size
	fsps.stats[op] = stats
}

func (fsps *FileSystemProfilerSource) GetProfileData() map[string]interface{} {
	fsps.mu.RLock()
	data := make(map[string]interface{})

	GIB := int64(1024 * 1024 * 1024) // Convert bytes to GiB
	for op, stats := range fsps.stats {
		data[op] = map[string]int64{
			"SequentialReadCount":     stats.SequentialReadCount,
			"RandomReadCount":         stats.RandomReadCount,
			"TotalAccessedFileHandle": stats.TotalAccessedFileHandle,
			"TotalAccessedInode":      stats.TotalAccessedInode,
			"TotalSizeReadAccessedGB": (stats.TotalSizeReadAccessed + GIB - 1) / (1024 * 1024 * 1024), // Convert bytes to GB
		}
	}
	fsps.mu.RUnlock()

	fsps.mu.Lock()
	defer fsps.mu.Unlock()
	fsps.stats = make(map[string]DataAccessStats) // Reset stats after getting profile data

	return data
}
