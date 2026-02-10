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
- `-dry-run`: Show what would be deleted
- `-debug`: Enable debug logging

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

## Example

```bash
./gcsfuse-scc-gc -cache-dir=/mnt/nfs-cache -target-size-mb=10240 -debug
```
Output:
```
time=2026-02-10T10:38:30.989Z level=INFO msg="Starting LRU cache eviction" cache_dir=/mnt/nfs-cache target_size_mb=1024
time=2026-02-10T10:38:31.244Z level=DEBUG msg="Manifest created" files=100 total_size_mb=800 scan_duration=126.255393ms
time=2026-02-10T10:38:31.244Z level=INFO msg="Cache below target, nothing to do" cache_size_mb=800 target_size_mb=1024
```

## Notes

- **Multi-host safe**: Can run concurrently on multiple hosts
- **Error tolerant**: Handles ENOENT/ENOTEMPTY gracefully
- **Parallel operations**: 10 concurrent file operations
- Works with SHA256-based cache structure: `cache-dir/XX/YY/hash/offset_offset.bin`
