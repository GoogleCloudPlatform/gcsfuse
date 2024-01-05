package log_parser

// StructuredLogEntry stores the structured format to be created from logs.
type StructuredLogEntry struct {
	Handle     int
	StartTime  int64
	ProcessID  int64
	InodeID    int64
	BucketName string
	ObjectName string
	// It can be safely assumed that the Chunks will be sorted on timestamp as logs
	// are parsed in the order of timestamps.
	Chunks []ChunkData
}

// ChunkData stores the format of chunk to be stored StructuredLogEntry.
type ChunkData struct {
	StartTime     int64
	StartOffset   int64
	Size          int64
	CacheHit      bool
	IsSequential  bool
	OpID          string
	ExecutionTime string
}

// HandleAndChunkIndex is used to store reverse mapping of FileCache operation id to
// Handle and chunk index stored in structure.
type HandleAndChunkIndex struct {
	Handle     int
	ChunkIndex int
}
