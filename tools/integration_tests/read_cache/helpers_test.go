package read_cache

import (
	"path"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

// Expected is a helper struct that stores list of attributes to be validated from logs.
type Expected struct {
	StartTimeStampSeconds int64
	EndTimeStampSeconds   int64
	BucketName            string
	ObjectName            string
	content               string
}

func readFileAndGetExpectedOutcome(testDirPath, fileName string, t *testing.T) *Expected {
	expected := &Expected{
		StartTimeStampSeconds: time.Now().Unix(),
		BucketName:            setup.TestBucket(),
		ObjectName:            path.Join(testDirName, setup.OnlyDirMounted(), fileName),
	}

	content, err := operations.ReadFileSequentially(path.Join(testDirPath, fileName), MiB)
	if err != nil {
		t.Errorf("Failed to read file in first iteration: %v", err)
	}
	expected.EndTimeStampSeconds = time.Now().Unix()
	expected.content = strings.Trim(string(content), "\x00")

	return expected
}

func validate(expected *Expected, logEntry *read_logs.StructuredReadLogEntry,
	isSeq, cacheHit bool, chunkCount int, t *testing.T) {
	if logEntry.StartTimeSeconds < expected.StartTimeStampSeconds {
		t.Errorf("start time in logs less than actual start time.")
	}
	if logEntry.BucketName != expected.BucketName {
		t.Errorf("Bucket names don't match! Expected: %s, Got from logs: %s",
			expected.BucketName, logEntry.BucketName)
	}
	if logEntry.ObjectName != expected.ObjectName {
		t.Errorf("Object names don't match! Expected: %s, Got from logs: %s",
			expected.ObjectName, logEntry.ObjectName)
	}
	if len(logEntry.Chunks) != chunkCount {
		t.Errorf("chunks read don't match! Expected: %d, Got from logs: %d",
			chunkCount, len(logEntry.Chunks))
	}
	if logEntry.Chunks[len(logEntry.Chunks)-1].StartTimeSeconds > expected.EndTimeStampSeconds {
		t.Errorf("end time in logs more than actual end time.")
	}
	if cacheHit != logEntry.Chunks[0].CacheHit {
		t.Errorf("Expected Cache Hit: %t, Got from logs: %t", cacheHit, logEntry.Chunks[0].CacheHit)
	}
	if isSeq != logEntry.Chunks[0].IsSequential {
		t.Errorf("Expected Is Sequential: %t, Got from logs: %t", isSeq, logEntry.Chunks[0].IsSequential)
	}
}

func (s *testStruct) validateFileInCacheDirectory(t *testing.T) {
	// Validate that the file is present in cache location.
	expectedPathOfCachedFile := path.Join(cacheLocationPath, cacheSubDirectoryName, setup.TestBucket(), testDirName, testFileName)
	fileInfo, err := operations.StatFile(expectedPathOfCachedFile)
	if err != nil {
		t.Errorf("Failed to find cached file %s: %v", expectedPathOfCachedFile, err)
	}
	// Validate file size in cache directory matches actual file size.
	if (*fileInfo).Size() != fileSize {
		t.Errorf("Incorrect cached file size. Expected %d, Got: %d", fileSize, (*fileInfo).Size())
	}
	// Validate content of file in cache directory matches GCS.
	content, err := operations.ReadFile(expectedPathOfCachedFile)
	if err != nil {
		t.Errorf("Failed to read cached file %s: %v", expectedPathOfCachedFile, err)
	}
	client.ValidateObjectContentsFromGCS(s.ctx, s.storageClient, testDirName, testFileName, string(content), t)
}
