package lru_test

import (
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
	. "github.com/jacobsa/ogletest"
)

// RadixCacheTest runs the exact same test suite as CacheTest but uses the radixCache
// implementation to ensure 1:1 behavioral parity.
type RadixCacheTest struct {
	CacheTest
}

func init() { RegisterTestSuite(&RadixCacheTest{}) }

func (t *RadixCacheTest) SetUp(*TestInfo) {
	locker.EnableInvariantsCheck()
	t.cache = lru.NewRadixCache(MaxSize)
}

// Override Test_EraseEntriesWithGivenPrefix_Concurrent to use NewRadixCache
func (t *RadixCacheTest) Test_EraseEntriesWithGivenPrefix_Concurrent() {
	t.CacheTest.Test_EraseEntriesWithGivenPrefix_Concurrent()
}
