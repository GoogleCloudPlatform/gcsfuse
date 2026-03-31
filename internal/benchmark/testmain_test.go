// Package benchmark — test entry point.
//
// TestMain runs before any test in this package.  It caps the write-pool RAM
// budget to a development-safe size so that unit tests never allocate the full
// production pool (16 GiB).
//
// Production runs (the actual gcs-bench binary) are unaffected: they never
// execute TestMain and always use poolBytesPerSide from constants.go.

package benchmark

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Limit the DoubleDataPool to 128 MiB per ring side (256 MiB total) for all
	// unit tests.  This is more than sufficient to exercise the pool pipeline
	// while keeping total test RSS well under 1 GiB regardless of object size.
	// Production uses poolBytesPerSide (8 GiB/side → 16 GiB total).
	overridePoolBytesPerSide = 128 << 20 // 128 MiB per side → 256 MiB total

	os.Exit(m.Run())
}
