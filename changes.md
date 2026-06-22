# Changes for Negative Cache TTL on Implicit Directories

## Summary

Added integration test coverage to verify that the negative stat cache correctly respects the negative TTL configuration when handling implicit directories in gcsfuse.

## Details

1. **New Integration Test File:**
   - Created `tools/integration_tests/negative_stat_cache/implicit_dir_finite_negative_stat_cache_test.go`
   - Added `implicitDirFiniteNegativeStatCacheTest` suite testing the `--implicit-dirs` behavior alongside finite negative stat cache configuration.
   - The test asserts that checking an implicit directory when it doesn't exist successfully populates the negative stat cache.
   - It simulates an implicit directory creation by adding an object behind the scenes and ensures it remains unseen (due to negative stat cache) until the 5-second TTL expires.

2. **Configuration Updates:**
   - Appended a new `ConfigItem` within `tools/integration_tests/negative_stat_cache/setup_test.go` for the negative stat cache tests.
   - The configuration runs the newly added test suite (`TestImplicitDirFiniteNegativeStatCacheTest`) with the flags `{"--metadata-cache-negative-ttl-secs=5", "--implicit-dirs"}`.
   - Set the compatibility configuration parameter `hns` to `false` because Hierarchical Namespace (HNS) buckets do not conceptually support implicit directories.

## Validation

- Verified that all unit tests correctly pass using `go test ./...`
- Specifically executed and successfully passed the added integration test with `go test ./tools/integration_tests/negative_stat_cache/... -run TestImplicitDirFiniteNegativeStatCacheTest`.