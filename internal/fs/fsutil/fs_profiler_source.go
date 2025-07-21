package fsutil

import (
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workloadprofiler"
	"sync"
)

type DataAccessStats struct {
	SequentialReadCount     int64  `yaml:"sequential_read_count"`
	RandomReadCount         int64  `yaml:"random_read_count"`
	TotalAccessedFileHandle int64  `yaml:"total_accessed_file_handle"`
	TotalAccessedInode      int64  `yaml:"total_accessed_inode"`
	TotalSizeReadAccessed   int64  `yaml:"total_size_read_accessed"`
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
			"sequential_read_count":     stats.SequentialReadCount,
			"random_read_count":         stats.RandomReadCount,
			"total_accessed_file_handle": stats.TotalAccessedFileHandle,
			"total_accessed_inode":      stats.TotalAccessedInode,
			"total_size_read_accessed_gb": (stats.TotalSizeReadAccessed + GIB - 1) / (1024 * 1024 * 1024), // Convert bytes to GB
		}
	}
	fsps.mu.RUnlock()

	fsps.mu.Lock()
	defer fsps.mu.Unlock()
	fsps.stats = make(map[string]DataAccessStats) // Reset stats after getting profile data

	return data
}
