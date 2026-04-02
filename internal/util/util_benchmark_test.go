package util

import (
	"testing"
)

func BenchmarkBytesToHigherMiBs(b *testing.B) {
	bytes := uint64(1048576) // 1 MiB
	for b.Loop() {
		_ = BytesToHigherMiBs(bytes)
	}
}
