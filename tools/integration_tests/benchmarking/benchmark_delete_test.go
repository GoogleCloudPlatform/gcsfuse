package benchmarking

import (
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/benchmark_setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

type benchmarkDeleteTest struct{}

func (s *benchmarkDeleteTest) SetupB(t *testing.B) {
	testDirPath = setup.SetupTestDirectory(testDirName)
}

func (s *benchmarkDeleteTest) TeardownB(t *testing.B) {}

// createFilesToDelete creates the below object in the bucket.
// benchmarking/a.txt
func createFilesToDelete(t *testing.B) {
	operations.CreateFileOfSize(5, path.Join(testDirPath, "a.txt"), t)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *benchmarkDeleteTest) Benchmark_Delete(t *testing.B) {
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		createFilesToDelete(t)
		if err := os.Remove(path.Join(testDirPath, "a.txt")); err != nil {
			t.Errorf("testing error: %v", err)
		}
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func Benchmark_Delete(t *testing.B) {
	ts := &benchmarkDeleteTest{}
	benchmark_setup.RunBenchmarks(t, ts)
}
