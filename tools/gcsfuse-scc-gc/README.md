# GCSFuse Shared Chunk Cache (SCC) Garbage Collector

Removes least recently used files from GCSFuse shared cache to maintain target size.

## Build

```bash
cd tools/gcsfuse-scc-gc
go build .
```

## Usage

```bash
./gcsfuse-scc-gc -cache-dir=/mnt/nfs-cache -target-size-mb=10240
```

**Options:**
- `-cache-dir` (required): Path to cache directory
- `-target-size-mb`: Target size in MB (default: 10240)
- `-concurrency`: Maximum concurrent file operations (default: 16)
- `-dry-run`: Show what would be deleted
- `-debug`: Enable debug logging

## File System Requirements

This garbage collector tool requires a file system that supports:

1. **Atomic rename**: The tool uses atomic rename operations to safely expire cache files (renaming `.bin` to `.bak`) without disrupting concurrent reads from multiple GCSFuse instances.

2. **Access time (atime) tracking**: The LRU eviction algorithm depends on file access times to determine which chunks are least recently used. The file system should track atime, either through:

**Note:** NFS and most POSIX-compliant file systems meet these requirements.

## Scheduling

### Cron (hourly)

```bash
crontab -e
# Add:
0 * * * * /usr/local/bin/gcsfuse-scc-gc -cache-dir=/mnt/nfs-cache -target-size-mb=10240 2>&1 | logger -t gcsfuse-scc-gc
```

### SystemD Timer

**Service:** `/etc/systemd/system/gcsfuse-scc-gc.service`
```ini
[Unit]
Description=GCSFuse Cache LRU Eviction

[Service]
Type=oneshot
ExecStart=/usr/local/bin/gcsfuse-scc-gc -cache-dir=/mnt/nfs-cache -target-size-mb=10240
```

**Timer:** `/etc/systemd/system/gcsfuse-scc-gc.timer`
```ini
[Unit]
Description=Run GCSFuse Cache LRU hourly

[Timer]
OnCalendar=hourly
RandomizedDelaySec=5min

[Install]
WantedBy=timers.target
```

**Enable:**
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now gcsfuse-scc-gc.timer
```

## How It Works

1. Cleans up any leftover `.bak` files from the previous runs.
2. Scans cache directory for `.bin` files with atime and size
3. If total size < target, then exit without expiration or eviction.
4. Sorts by atime and selects oldest files to expire.
5. Renames selected files to `.bak` (kept until next run for ongoing reads).
6. Removes old `.tmp` files (older than 1 hour)
7. Cleans up empty directories.

**Two-phase eviction:** Files are renamed to `.bak` on the current run and deleted on the next run. This ensure no
concurrent read and eviction of chunked file.

## Cache Directory Structure

The tool expects the cache to be organized in a `gcsfuse-shared-chunk-cache` subdirectory within the specified cache directory:
- Format: `<cache-dir>/gcsfuse-shared-chunk-cache/<2-char>/<2-char>/<full-hash>/<start>_<end>.bin`
- Automatically detects and uses this subdirectory structure
- Uses SHA256-based hashing for cache organization

**Note:** This is separate from the regular GCSFuse file cache which uses `gcsfuse-file-cache` subdirectory.

## Example

```bash
./gcsfuse-scc-gc -cache-dir=/mnt/nfs-cache -target-size-mb=10240 -debug
```
Output:
```
time=2026-02-12T19:06:36.237Z level=INFO msg="Starting LRU cache eviction" cache_dir=/mnt/nfs-cache target_size_mb=10240
time=2026-02-12T19:06:36.640Z level=DEBUG msg="Manifest created" files=100 total_size_mb=800 scan_duration=120.731382ms
time=2026-02-12T19:06:36.640Z level=INFO msg="Cache below target, nothing to do" cache_size_mb=800 target_size_mb=10240
```
