# GCSFuse Shared Cache LRU Eviction Tool

Removes least recently used files from GCSFuse shared cache to maintain target size.

## Build

```bash
cd tools/cache_lru
go build -o gcsfuse-shared-cache-lru
```

## Usage

```bash
./gcsfuse-shared-cache-lru -cache-dir=/mnt/nfs-cache -target-size-mb=10240
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
0 * * * * /usr/local/bin/gcsfuse-shared-cache-lru -cache-dir=/mnt/nfs-cache -target-size-mb=10240 2>&1 | logger -t gcsfuse-lru
```

### SystemD Timer

**Service:** `/etc/systemd/system/gcsfuse-cache-lru.service`
```ini
[Unit]
Description=GCSFuse Cache LRU Eviction

[Service]
Type=oneshot
ExecStart=/usr/local/bin/gcsfuse-shared-cache-lru -cache-dir=/mnt/nfs-cache -target-size-mb=10240
```

**Timer:** `/etc/systemd/system/gcsfuse-cache-lru.timer`
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
sudo systemctl enable --now gcsfuse-cache-lru.timer
```

## How It Works

1. Cleans up any leftover `.bak` files from previous runs
2. Scans cache directory for `.bin` files with atime and size
3. If total size < target, exits
4. Sorts by atime and selects oldest files to evict
5. Renames selected files to `.bak` (kept until next run for safety)
6. Removes old `.tmp` files (older than 1 hour)
7. Cleans up empty directories

**Two-phase eviction:** Files are renamed to `.bak` on the current run and deleted on the next run. This ensure no
concurrent read and eviction of chunked file.

## Example

```bash
./gcsfuse-shared-cache-lru -cache-dir=/mnt/nfs-cache -target-size-mb=10240 -verbose
```
Output:
```
Manifest created: 5432 files, 15360 MB total (scan took 2.3s)
Need to evict 5120 MB (2145 files)
LRU cache eviction completed
```

## Notes

- **Multi-host safe**: Can run concurrently on multiple hosts
- **Error tolerant**: Handles ENOENT/ENOTEMPTY gracefully
- **Parallel operations**: 10 concurrent file operations
- Works with SHA256-based cache structure: `cache-dir/XX/YY/hash/offset_offset.bin`
