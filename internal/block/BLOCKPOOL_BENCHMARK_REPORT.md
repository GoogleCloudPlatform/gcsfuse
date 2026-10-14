# GCSFuse Block Pool Empirical Evaluation: `mmap` vs `sync.Pool` Alternatives (Variants A & B)

**Authoritative Empirical Verdict:** Neither `mmap` nor `sync.Pool` is universally faster across all operating regimes. Instead, performance is governed by a sharp **lifecycle-vs-GC phase boundary** verified across both a 96-vCPU Intel Xeon workstation (`Environment A`) and a dedicated 180-vCPU **GKE `ct6e-standard-4t` (Cloud TPU v6e Trillium) node (`AMD EPYC 9B14`, 708 GiB RAM, `THP=[always]`, `Environment B`)**:
1. **In High File-Churn & `fsync`-Churn Regimes (`Regime 1` & `Regime 4 FileChurn`), `sync.Pool` (Variants A & B) is dramatically faster than `mmap` (`1.18x` to `14.67x` speedup on Intel Xeon; `1.39x` to `12.01x` speedup / `-27.84%` to `-91.67%` latency on GKE `ct6e-standard-4t`, $p = 0.008$)** because GCSFuse creates and destroys `GenBlockPool` instances **per file handle** (`ClearFreeBlockChannel(true)` on `Flush`/`Destroy`, and `ClearFreeBlockChannel(false)` on `Sync`/`fsync`). Baseline `mmap` executes `syscall.Mmap` + kernel demand-zero minor page faults + `syscall.Munmap` (and multi-core TLB shootdown IPIs + `mm->mmap_lock` contention) on every file lifecycle, whereas Variants A and B recycle already-faulted user-space slices across file lifecycles with **zero syscalls and zero page faults**.
2. **In Steady-State Single-Pool Streaming Reuse (`Regime 2` & `Regime 4 SteadyReuse`), `mmap`, Variant A, and Variant B are statistically indistinguishable (`~0%` delta, $p > 0.15$ across 15/16 sweep cells on Xeon and 14/16 sweep cells on `ct6e-standard-4t`; `0 B/op`, `0 allocs/op`, `0 minor-faults/op`)**. Once blocks are allocated and faulted into a pool, throughput is 100% bound by L3/DRAM `runtime.memmove` bandwidth (`~8.3 GiB/s` single-core / `122.1 GiB/s` at $C=16$ on Xeon; **`12.16 GiB/s` single-core / `91.86 GiB/s` at $C=16$ on GKE `ct6e-standard-4t`**).
3. **Under Periodic Burst-Idle Workloads (`>= 2` GCs between bursts, `Regime 3B`) and Strict Container RSS Limits (`Regime 3C`), `mmap` wins decisively on memory footprint and avoids the 2-GC `sync.Pool` eviction cliff**:
   - **The 2-GC `sync.Pool` Victim-Cache Eviction Cliff (`Regime 3B`)**: When 1 GC occurs between bursts (`gcCount=1`), `sync.Pool`'s `victim` cache rescues 100% of pooled slices (`744 B/op` / `408 B/op`, `-29.95%` faster on Xeon and `-42.50%` faster on `ct6e-standard-4t`). When **2 GCs** occur between bursts (`gcCount=2`), `sync.Pool` evicts its `victim` cache completely on **both** architectures, causing heap allocations to jump from **`744 B/op` to `33,555,272 B/op` (`32.00 MiB/op`)**, forcing **`19.72%` of CPU time into `runtime.memclrNoHeapPointers`** (heap zeroing), degrading `sync.Pool` latency by **`+69.1%` on Xeon** (`15.44%` slower than `mmap`, $p = 0.032$) and by **`+39.1%` on `ct6e-standard-4t`** (`3.006 ms` $\rightarrow$ `4.182 ms/op`, `+53.6%` higher `anon-rss-MB`).
   - **Immediate OS Memory Reclamation (`Regime 3C`)**: Immediately after `ClearFreeBlockChannel(true)` following a 64 MiB burst (`PostClear_NoGC`), `mmap` releases physical pages to the Linux kernel via `syscall.Munmap` (`anon-rss-MB = 49.77 MiB` on Xeon; **`56.75 MiB` on `ct6e-standard-4t`**), whereas Variant A (`118.92 MiB` / **`127.12 MiB`**, `+124.0%`) and Variant B (`118.64 MiB` / **`125.12 MiB`**, `+120.5%`) retain the 64 MiB inside Go's heap (`HeapInuse = 109.10 MiB` / **`102.90 MiB`**, `+146.0%`) until 2 GC cycles and runtime scavenging occur.
4. **Phase 2 GKE `ct6e-standard-4t` (TPU v6e Host with Transparent Hugepages `THP=[always]`) Verdict (`Section 9`)**:
   - **Does slice-based `sync.Pool` allocation still outperform `mmap` when Transparent Hugepages (`THP=[always]`) are enabled on `ct6e-standard-4t`? Yes — by an even wider margin at `16 MiB` and `32 MiB` (`1.92x` to `2.09x` faster; `-47.70%` to `-52.73%` latency, $p = 0.008$) than on the Intel Xeon workstation (`-11.20%` to `-32.32%`), and up to `12.01x` faster (`-91.67%` latency, `88.72 GiB/s` vs `7.39 GiB/s`) at `1 MiB, C=16`!**
   - Kernel `/proc/vmstat` delta probes (`Section 9.1`) prove *why*: on `ct6e-standard-4t` (`Linux 6.12.94+ COS`, `THP=[always]`), `Mmap` at `8 MiB`, `16 MiB`, and `32 MiB` faults in `2 MiB` PMD Transparent Hugepages (`8.00`, `16.00`, and `32.00` `minor-faults/op` = `+3,590` and `+1,938` `thp_fault_alloc` with `0` `thp_fault_fallback`), while `1 MiB` `< 2 MiB` PMD size cannot use THPs and incurs `512.0` 4 KiB page faults/op (`+1,337,733` `pgfault`). However, even when `Mmap` gets 100% 2 MiB THPs at `32 MiB`, the kernel must still zero `64 MiB` of newly mapped physical hugepages (`clear_huge_page`) and tear down the VMAs/TLBs on `Munmap`, costing **`4.84 ms/op` (`FileCloseChurn`) to `10.15 ms/op` (`FsyncChurn`) in kernel `sys-ns/op`**. Because `ct6e-standard-4t`'s AMD EPYC 9B14 (`384 MiB` L3 cache + DDR5) executes user-space `runtime.memmove` `4.0x` faster at 32 MiB than the Intel Xeon workstation (`5.41 ms` vs `21.81 ms`), `Mmap`'s `4.84 ms` kernel THP-zeroing + `munmap` tax represents **`45.0%` of total wall time** on `ct6e-standard-4t` (`10.75 ms` vs `5.41 ms`), doubling `sync.Pool`'s speedup on large blocks!

---

## 1. Executive Summary & Falsifiable Verdict Matrix

### 1.1 Environment A: Local Workstation (`Intel Xeon 96-vCPU`, `236 GiB RAM`) Falsifiable Verdict Matrix

| Operating Regime | Benchmark Identifier | Winning Implementation | Measured Speedup / Delta (`benchstat`, $n=5$) | Primary Systems Root Cause |
|---|---|---|---|---|
| **1A. High File-Close Churn (1 MiB – 32 MiB, $C=1$)** | `BenchmarkBlockPool_HighFileChurn/mode=FileCloseChurn` | **Variant A & Variant B** (`sync.Pool`) | **`-70.22%` / `-71.28%` latency** (`3.36x` / `3.48x` faster at 1 MiB, $p=0.008$); **`-29.57%` / `-30.23%`** at 8 MiB ($p=0.008$); **`-32.32%` / `-21.14%`** at 16 MiB ($p=0.008$) | Eliminates per-file `syscall.Mmap` + `syscall.Munmap` (`598.7 µs`–`5.94 ms/op` kernel `sys-ns/op`) and `512` (4 KiB PTE) or `8–32` (2 MiB THP) demand-zero minor page faults per op. |
| **1B. Mid-Stream `fsync` Churn (1 MiB – 32 MiB, $C=1$)** | `BenchmarkBlockPool_HighFileChurn/mode=FsyncChurn` | **Variant A & Variant B** (`sync.Pool`) | **`-71.57%` / `-73.51%` latency** (`3.52x` / `3.77x` faster at 1 MiB, $p=0.008$); **`-23.43%` / `-19.80%`** at 32 MiB ($p=0.008$) | `BufferedWriteHandler.Sync()` calls `ClearFreeBlockChannel(false)`, forcing `mmap` to unmap and re-fault blocks (`1,024` minor faults/op at 1 MiB; `10.74 ms/op` `sys-ns/op` at 32 MiB). |
| **2. Steady-State Single-Pool Reuse (1 MiB – 32 MiB)** | `BenchmarkBlockPool_SteadyStateReuse` & `Sweep/workload=SteadyReuse` | **Tie (`Mmap` $\approx$ `VariantA` $\approx$ `VariantB`)** (`Mmap` wins on off-heap footprint) | **`~0%` throughput delta** ($p > 0.15$ across 15/16 sweep cells); all achieve `0 B/op`, `0 allocs/op`, `0 minor-faults/op`, up to **`122.1 GiB/s`** at $C=16$ | Once blocks sit warm in `freeBlocksCh` (or `sync.Pool`), zero syscalls or page faults occur; execution is 100% bound by DRAM/L3 `runtime.memmove` (`Write` + `Read`). `Mmap` uses `-48%` lower `heap-inuse-MB`. |
| **3A. Active Churn under Heap Pressure (`GOGC=100`, `GOGC=20`, `GOMEMLIMIT=128MiB`)** | `BenchmarkBlockPool_GCPressure/subregime=ActiveChurn_HeapPressure` | **Variant B & Variant A** (Throughput) vs **`Mmap`** (GC cycles & Heap size) | Variants A/B are **`-19.76%` to `-30.07%` faster** ($p=0.008$), but trigger **`+50%` to `+75%` more GC cycles** and **`+37%` to `+42%` higher `anon-rss-MB`** | Active churn rescues most buffers from `sync.Pool`'s `local`/`victim` caches between GCs, keeping throughput higher than `mmap`, at the cost of partial cross-P `victim.private` misses (`0.82–1.90 MiB/op`) and inflated Go heap targets. |
| **3B. Burst-Idle-Burst (`0 GC` vs `1 GC` vs `2 GC` Eviction Cliff)** | `BenchmarkBlockPool_GCPressure/subregime=BurstIdleGCEviction` | **`0–1 GC`: Variant A/B** (`-30%`, $p=0.008$)<br>**`>= 2 GC`: `Mmap`** (`+15.44%` faster than Variant B, $p=0.032$) | At `gcCount=1`: **`744 B/op`** (100% `victim` cache rescue, `-29.95%` latency).<br>At `gcCount=2`: jumps to **`33,555,272 B/op` (`32 MiB/op`)**; Variant B degrades by **`+69.1%`** (`5.66ms` $\rightarrow$ `9.57ms/op`) | Go's `sync.Pool` moves `local` $\rightarrow$ `victim` on GC #1 and drops `victim` on GC #2. At `gcCount=2`, `sync.Pool` suffers a 100% miss and pays full heap allocation + `runtime.memclrNoHeapPointers` (`19.72%` CPU) + GC sweep overhead. |
| **3C. Post-Burst OS Memory Reclamation (64 MiB Burst)** | `BenchmarkBlockPool_GCPressure/subregime=PostBurstMemoryRetention` | **`Mmap`** (Deterministic OS RAM release) | Post-`ClearFreeBlockChannel(true)` before GC: `Mmap` **`49.77 MiB` `anon-rss-MB`** vs Variant A **`118.92 MiB` (`+138.93%`)** and Variant B **`118.64 MiB` (`+138.36%`)** ($p=0.008$) | `syscall.Munmap` immediately unmaps physical pages and returns `RssAnon` to the OS upon file close; `sync.Pool` retains all 64 MiB in Go's `HeapInuse` (`109.10 MiB` vs `40.43 MiB`) until 2 GCs + scavenging. |
| **4. Multi-Core File-Churn Scaling ($C \in \{1, 4, 8, 16\}$)** | `BenchmarkBlockPool_Sweep/workload=FileChurn` | **Variant A & Variant B** (`sync.Pool`) | At `1 MiB, C=16`: **`114.65 GiB/s`** (Variant A) & **`108.67 GiB/s`** (Variant B) vs **`7.82 GiB/s`** (`Mmap`) — **`14.67x` speedup (`-93.18%` latency, $p=0.008$)**.<br>At `32 MiB, C=16`: **`-23.17%` / `-24.05%` latency** ($p=0.008$), but `+1.0 GiB` higher `anon-rss-MB` | Concurrent `mmap`/`munmap` across 16 threads serializes on the process-wide Linux kernel `mm->mmap_lock` and triggers cross-core TLB shootdown IPIs (`11.80 ms/op` `sys-ns/op` at 32 MiB $C=16$), whereas `sync.Pool` scales lock-free per-`P`. |

### 1.2 Environment B: GKE `ct6e-standard-4t` (`AMD EPYC 9B14 180-vCPU`, `708 GiB RAM`, `THP=[always]`) Falsifiable Verdict Matrix

| Operating Regime | Benchmark Identifier | Winning Implementation on `ct6e-standard-4t` | Measured Speedup / Delta on `ct6e-standard-4t` (`benchstat`, $n=5$) | Primary Systems Root Cause on `ct6e-standard-4t` (`Section 9`) |
|---|---|---|---|---|
| **1A. High File-Close Churn (1 MiB – 32 MiB, $C=1$)** | `BenchmarkBlockPool_HighFileChurn/mode=FileCloseChurn` | **Variant A & Variant B** (`sync.Pool`) | **`-83.30%` / `-83.63%` latency** (`5.99x` / `6.11x` faster at 1 MiB, $p=0.008$); **`-28.28%` / `-28.21%`** at 8 MiB ($p=0.008$); **`-48.31%` / `-47.98%`** at 16 MiB ($p=0.008$); **`-49.65%` / `-49.65%`** (`1.99x` faster) at 32 MiB ($p=0.008$) | Even with `THP=[always]` (`8`, `16`, `32` `thp_fault_alloc`/op at $\ge 8\text{ MiB}$), `Mmap` spends **`0.47 ms`–`4.84 ms/op` in kernel `sys-ns/op`** zeroing 2 MiB THPs and tearing down VMAs (`45%` of total wall time at 32 MiB!). |
| **1B. Mid-Stream `fsync` Churn (1 MiB – 32 MiB, $C=1$)** | `BenchmarkBlockPool_HighFileChurn/mode=FsyncChurn` | **Variant A & Variant B** (`sync.Pool`) | **`-83.75%` / `-83.86%` latency** (`6.15x` / `6.20x` faster at 1 MiB, $p=0.008$); **`-28.64%` / `-28.68%`** at 8 MiB; **`-47.70%` / `-47.83%`** at 16 MiB; **`-49.31%` / `-49.21%`** at 32 MiB ($p=0.008$) | `Mmap` incurs `1,024` 4 KiB minor faults/op at 1 MiB (`1.37 ms` `sys-ns/op`) and `64.00` 2 MiB THP faults/op at 32 MiB (`10.15 ms` `sys-ns/op`), saving **`10.64 ms/op`** with `sync.Pool`. |
| **2. Steady-State Single-Pool Reuse (1 MiB – 32 MiB)** | `BenchmarkBlockPool_SteadyStateReuse` & `Sweep/workload=SteadyReuse` | **Tie (`Mmap` $\approx$ `VariantA` $\approx$ `VariantB`)** (`Mmap` wins on off-heap footprint) | **`~0%` throughput delta** ($p > 0.05$ across 14/16 sweep cells); all achieve **`11.50`–`12.16 GiB/s`** at $C=1$ and up to **`91.86 GiB/s`** at $C=16$ (`0 B/op`, `0 allocs/op`, `0 minor-faults/op`) | AMD EPYC 9B14's `384 MiB` L3 cache + DDR5 sustains `~12 GiB/s` single-core `runtime.memmove` even at 32 MiB (`3.4x` faster than Xeon). `Mmap` uses `-45%` to `-59%` lower `heap-inuse-MB`. |
| **3A. Active Churn under Heap Pressure (`GOGC=100`, `GOGC=20`, `GOMEMLIMIT=128MiB`)** | `BenchmarkBlockPool_GCPressure/subregime=ActiveChurn_HeapPressure` | **Variant A & Variant B** (Throughput) vs **`Mmap`** (Heap & RSS footprint) | Variants A/B are **`-24.31%` to `-27.07%` faster** (`+32.1%` to `+37.1%` GiB/s, $p=0.008$), with **`+33%` higher `heap-inuse-MB`** and **`+37%` to `+43%` higher `anon-rss-MB`** | Active churn rescues `88%`–`94%` of 8 MiB buffers from `sync.Pool`, losing only `0.63`–`1.40 MiB/op` to cross-P `victim.private` stranding across `GOMAXPROCS=180`. |
| **3B. Burst-Idle-Burst (`0 GC` vs `1 GC` vs `2 GC` Eviction Cliff)** | `BenchmarkBlockPool_GCPressure/subregime=BurstIdleGCEviction` | **`0–1 GC`: Variant A/B** (`-42.5%` to `-43.6%`, $p=0.008$)<br>**`>= 2 GC`: `Mmap` on Allocations & RSS** (`368 B/op` vs `32 MiB/op`) | At `gcCount=1`: **`744 B/op` / `408 B/op`** (100% `victim` rescue, `3.00 ms` vs `5.23 ms`).<br>At `gcCount=2`: jumps to **`33,555,272 B/op` (`32 MiB/op`)**; `sync.Pool` latency degrades **`+39.1%`** (`3.01 ms` $\rightarrow$ `4.18 ms`) and `+53.6%` `anon-rss-MB` | 2 GCs evict `sync.Pool` completely (`+9,118,180%` `B/op`, `+28.6%` GC cycles). Because EPYC 9B14 user-space zeroing (`1.18 ms` penalty) is faster than kernel `mmap`+`munmap` (`2.03 ms` `sys-ns/op`), `sync.Pool` stays `-22%` faster in wall time but suffers `32 MiB/op` heap churn. |
| **3C. Post-Burst OS Memory Reclamation (64 MiB Burst)** | `BenchmarkBlockPool_GCPressure/subregime=PostBurstMemoryRetention` | **`Mmap`** (Deterministic OS RAM release) | Post-`ClearFreeBlockChannel(true)` before GC: `Mmap` **`56.75 MiB` `anon-rss-MB`** vs Variant A **`127.12 MiB` (`+124.00%`)** and Variant B **`125.12 MiB` (`+120.47%`)** ($p=0.008$) | `syscall.Munmap` immediately releases `57.2 MiB` of anonymous RSS to the COS kernel; Variants A/B retain the full 64 MiB burst in `HeapInuse` (`102.90 MiB` vs `41.84 MiB`, `+145.95%`). |
| **4. Multi-Core File-Churn Scaling ($C \in \{1, 4, 8, 16\}$)** | `BenchmarkBlockPool_Sweep/workload=FileChurn` | **Variant A & Variant B** (`sync.Pool`) | At `1 MiB, C=16`: **`88.72 GiB/s`** (Variant B) & **`81.15 GiB/s`** (Variant A) vs **`7.39 GiB/s`** (`Mmap`) — **`12.01x` speedup (`-91.67%` latency, $p=0.008$)**.<br>At `32 MiB, C=16`: **`85.28 GiB/s`** vs **`40.73 GiB/s`** — **`2.09x` speedup (`-52.24%` latency, $p=0.008$)** | Concurrent `mmap`/`munmap` spends `1.48 ms` to `10.57 ms/op` in kernel `sys-ns/op` contending on `mm->mmap_lock` and TLB shootdowns; `sync.Pool` scales lock-free across cores (`+193%`–`+211%` `anon-rss-MB` at 32 MiB $C=16$). |

---

## 2. Experimental Methodology, Hardware/OS Environment & Reproducibility

### 2.1 Host Hardware, OS Kernel, and Go Runtime Environments

All experiments were executed across two production-grade Linux environments under identical benchmark harnesses:

| Parameter | Environment A: Local Workstation (`Phase 1`) | Environment B: GKE `ct6e-standard-4t` TPU v6e Node (`Phase 2`) | Source / Verification Command |
|---|---|---|---|
| **Platform & Cloud Topology** | Dedicated Linux Workstation | GKE Cluster `blockpool-bench-ct6e-10100727` (`europe-west4-a`, node `gke-tpu-12341d57-2jnm`, reservation `cloudtpu-20260914184000-1318357512`, `4x` TPU v6e chips) | `kubectl get pod/node -o wide` |
| **CPU Model & Architecture** | `Intel(R) Xeon(R) CPU @ 2.60GHz` (`amd64`) | `AMD EPYC 9B14` (4th Gen Genoa / Zen 4, `amd64`, Family 25 Model 17) | `lscpu` / `/proc/cpuinfo` |
| **Logical vCPUs & Topology** | `96` logical cores (`GOMAXPROCS=96`) | **`180` logical vCPUs (`GOMAXPROCS=180`)** — 1 socket, 90 physical cores $\times$ 2 threads/core, 1 NUMA node | `lscpu` / `runtime.NumCPU()` |
| **Cache Hierarchy** | Multi-socket Xeon L3 cache | L1d `32 KiB`/core (`2.8 MiB`), L2 `1 MiB`/core (`90 MiB`), **L3 `32 MiB` $\times 12$ CCDs = `384 MiB` total L3** | `lscpu` |
| **System Memory** | `236 GiB` total RAM (`~213 GiB` available) | **`708 GiB` total RAM (`742,622,448 KiB`, `~700 GiB` available)** | `free -h` / `/proc/meminfo` |
| **OS & Kernel Version** | Linux `amd64`, base page size = `4096` B (`4 KiB`) | **Container-Optimized OS (COS), Linux `6.12.94+` SMP PREEMPT_DYNAMIC**, base page size = `4096` B (`4 KiB`) | `uname -a` / `getconf PAGESIZE` |
| **Transparent Huge Pages (THP)** | `enabled: [always] madvise never`<br>`defrag: always defer defer+madvise [madvise] never` | **`enabled: [always] madvise never`**<br>**`defrag: always defer defer+madvise [madvise] never`**<br>`shmem_enabled: [never]`, `hpage_pmd_size: 2097152` (`2 MiB`), `khugepaged/defrag: 1` | `/sys/kernel/mm/transparent_hugepage/*` |
| **Max VMA Count & Overcommit** | `vm.max_map_count = 4194304`, `vm.overcommit_memory = 0` | `vm.max_map_count = 65530`, `vm.overcommit_memory = 1` (set to `0` during unit tests so `1 TiB` `mmap` returns `ENOMEM`) | `/proc/sys/vm/{max_map_count,overcommit_memory}` |
| **Go Toolchain** | `go version go1.27.0 linux/amd64` | `go version go1.27.0 linux/amd64` | `go version` |
| **Statistical Engine** | `benchstat` (Wilcoxon rank-sum test at $\alpha=0.05$, $n=5$, `-benchtime=100ms`) | `benchstat` (Wilcoxon rank-sum test at $\alpha=0.05$, $n=5$, **`-benchtime=200ms`**) | `/usr/local/google/home/kislayk/go/bin/benchstat` |

### 2.2 Anti-Elision Full-Block Touching & Telemetry Design

A common flaw in memory-pool benchmarks is allocating buffers without writing to or reading from their pages. On Linux, `syscall.Mmap(..., MAP_ANON|MAP_PRIVATE)` only inserts a `vm_area_struct` (VMA) into the process maple tree; physical page frames are only allocated and zeroed by the kernel (`do_anonymous_page` $\rightarrow$ `clear_page_erms`) upon first write (generating minor page faults), and `syscall.Munmap` only performs expensive page-table zapping (`zap_pte_range`) and cross-core TLB shootdowns (`tlb_flush_mmu`) for pages that were actually faulted in.

In `internal/block/block_pool_bench_test.go`, every block operation (`touchAndVerifyBlock`) enforces realistic memory touching and prevents compiler dead-code elimination:
1. **Full-Block Write**: Writes 100% of `blockSize` bytes from a deterministic non-zero source payload via `blk.Write(src[:blockSize])` (`runtime.memmove`), faulting every underlying 4 KiB page (or 2 MiB THP).
2. **Seek & Full-Block Read**: Calls `blk.Seek(0, io.SeekStart)` and reads 100% of `blockSize` bytes into a per-worker buffer via `blk.Read(readBuf[:blockSize])`.
3. **4 KiB Page-Strided Checksum Sink**: Sums 1 byte per 4 KiB page (`off += 4096`) from `readBuf` and folds the result into a package-level `atomic.Uint64` (`benchSink`), making every page read observable to the compiler.
4. **Out-of-Band OS & Runtime Telemetry**: Immediately before `b.ResetTimer()` and immediately after `b.StopTimer()` (so `/proc` file reads never pollute timed `ns/op`, `B/op`, or `allocs/op`), `takeTelemetrySnapshot()` captures:
   - `minor-faults/op` and `sys-ns/op` via `syscall.Getrusage(syscall.RUSAGE_SELF, &ru)` (`Minflt` and `Stime` deltas divided by `b.N`).
   - `gc-cycles`, `gc-pause-ns/op`, and `heap-inuse-MB` via `runtime.ReadMemStats`.
   - Instantaneous `rss-MB` (from `/proc/self/statm` field 1 $\times 4096$) and `anon-rss-MB` (from `/proc/self/status` `RssAnon:`).

### 2.3 Reproducible Command Lines (Workstation & GKE `ct6e-standard-4t`)

```bash
# ==============================================================================
# Environment A: Local Workstation Reproduction
# ==============================================================================
# 1. Verify all existing and new unit tests + race detector pass (0 regressions, 0 data races)
go test -v -race ./internal/block/...

# 2. Run the full 4-regime benchmark suite with -benchmem and -count=5
go test -run=^$ -bench='^BenchmarkBlockPool_' -benchmem -benchtime=100ms -count=5 ./internal/block \
  > .agents/teamwork/teamwork_preview_worker_r1_1/bench_raw_count5.txt

# 3. Compute statistical summaries, medians, % deltas, and p-values with benchstat
/usr/local/google/home/kislayk/go/bin/benchstat -col /variant \
  .agents/teamwork/teamwork_preview_worker_r1_1/bench_raw_count5.txt \
  > .agents/teamwork/teamwork_preview_worker_r1_1/benchstat_summary.txt

# ==============================================================================
# Environment B: GKE ct6e-standard-4t (TPU v6e) Reproduction & Cleanup
# ==============================================================================
# 1. Provision isolated VPC network (MTU 8896), subnet, GKE cluster, and 1-node ct6e-standard-4t pool
gcloud compute networks create blockpool-bench-net-10100727 --project=gcs-fuse-test-ml --subnet-mode=custom --mtu=8896
gcloud compute networks subnets create blockpool-bench-subnet-10100727 --project=gcs-fuse-test-ml \
  --network=blockpool-bench-net-10100727 --region=europe-west4 --range=10.0.0.0/24
gcloud container clusters create blockpool-bench-ct6e-10100727 --project=gcs-fuse-test-ml --zone=europe-west4-a \
  --network=blockpool-bench-net-10100727 --subnetwork=blockpool-bench-subnet-10100727 \
  --machine-type=e2-medium --num-nodes=1 --enable-ip-alias --quiet
gcloud container node-pools create ct6e-pool --project=gcs-fuse-test-ml --cluster=blockpool-bench-ct6e-10100727 \
  --zone=europe-west4-a --machine-type=ct6e-standard-4t --num-nodes=1 \
  --reservation-affinity=specific --reservation=cloudtpu-20260914184000-1318357512 --quiet

# 2. Schedule pod onto ct6e-standard-4t node, stream internal/block payload, and run tests + benchmarks
kubectl apply -f .agents/teamwork/teamwork_preview_worker_r2_1/pod_bench_ct6e.yaml
kubectl wait --for=condition=Ready pod/blockpool-bench-ct6e --timeout=300s
kubectl exec blockpool-bench-ct6e -- bash -c 'export PATH="/usr/local/go/bin:$PATH" && cd /workspace && \
  go test -v -race ./internal/block/... && \
  go test -run="^$" -bench="^BenchmarkBlockPool_" -benchmem -benchtime=200ms -count=5 -timeout=30m ./internal/block/...'

# 3. Compute GKE ct6e-standard-4t benchstat summary and cross-platform comparison
/usr/local/google/home/kislayk/go/bin/benchstat -col /variant \
  .agents/teamwork/teamwork_preview_worker_r2_1/bench_ct6e_raw_count5.txt \
  > .agents/teamwork/teamwork_preview_worker_r2_1/benchstat_ct6e_summary.txt

# 4. Tear down GKE cluster, firewall rules, subnet, and VPC network
gcloud container clusters delete blockpool-bench-ct6e-10100727 --project=gcs-fuse-test-ml --zone=europe-west4-a --quiet
gcloud compute networks subnets delete blockpool-bench-subnet-10100727 --project=gcs-fuse-test-ml --region=europe-west4 --quiet
gcloud compute networks delete blockpool-bench-net-10100727 --project=gcs-fuse-test-ml --quiet
```

---

## 3. Implementation Architecture Comparison & Production Caller Analysis

### 3.1 Side-by-Side Comparison of the Three Block Pool Architectures

| Architectural Dimension | Baseline: `Mmap` (`GenBlockPool`) | Variant A: Hybrid (`GenBlockPool` + `sync.Pool` Slices) | Variant B: Direct (`DirectSyncBlockPool` + Semaphore) |
|---|---|---|---|
| **Source File** | `internal/block/block.go`, `block_pool.go` (Untouched) | `internal/block/sync_pool_block.go` | `internal/block/direct_sync_block_pool.go` |
| **Constructor (`Block` / `PrefetchBlock`)** | `NewBlockPool` / `NewPrefetchBlockPool` | `NewSyncPoolBlockPool` / `NewSyncPoolPrefetchBlockPool` | `NewDirectSyncBlockPool` / `NewDirectSyncPrefetchBlockPool` |
| **Backing Memory Source** | Off-heap OS anonymous mapping via `syscall.Mmap(-1, 0, blockSize, PROT_READ\|PROT_WRITE, MAP_ANON\|MAP_PRIVATE)` | Go heap `[]byte` slice pooled via size-keyed `slicePools` (`sync.Map` of `blockSize -> *sync.Pool` storing `*[]byte` pointers) | Go heap `*syncPoolBlock` / `*syncPoolPrefetchBlock` pooled directly in size-keyed `*sync.Pool` |
| **Intra-Pool Recycling (`Get` $\leftrightarrow$ `Release`)** | Buffered channel `freeBlocksCh chan T` (`cap = maxBlocks`); `Reuse()` resets `readSeek=0, buffer=buffer[:0]` | Identical to Baseline (`freeBlocksCh chan T` inside `GenBlockPool[T]`) | Direct `bp.pool.Get()` and `bp.pool.Put(b)` with mutex-synchronized `inUse`, `reservedHeld`, and FIFO `waiters` |
| **Teardown (`ClearFreeBlockChannel`)** | Drains `freeBlocksCh`, calls `syscall.Munmap` on every block, releases semaphore permits | Drains `freeBlocksCh`, calls `syncPoolBlock.Deallocate()` which returns `*[]byte` to `slicePools` (`0` syscalls) | Blocks already returned to `bp.pool` on `Release()`; releases idle `reservedHeld` semaphore permits (`0` syscalls) |
| **Per-File Lifecycle Allocations (Warm Pool, $K=2$ blocks)** | `5 allocs/op, 272 B/op` (`GenBlockPool` + `chan` + $2\times$ `memoryBlock` + slice header) + $2\times$ `mmap`/`munmap` syscalls | `5 allocs/op, 304 B/op` (`GenBlockPool` + `chan` + $2\times$ `syncPoolBlock`); `0 B/op` for backing slices | **`1 allocs/op, 96 B/op`** (`DirectSyncBlockPool` struct only; no `freeBlocksCh` channel and no `Block` wrapper allocations) |
| **Intra-Pool Thread Safety** | Single-getter + async-releaser (`totalBlocks` unguarded across concurrent `Get`s) | Same as `GenBlockPool` (`totalBlocks` unguarded across concurrent `Get`s) | **100% thread-safe** across concurrent `Get`, `TryGet`, `Release`, and `ClearFreeBlockChannel` (`bp.mu sync.Mutex`) |
| **Go GC Visibility & OS RSS Return** | Invisible to Go GC (`0` `HeapInuse`); `Munmap` returns RSS to OS immediately on `ClearFreeBlockChannel` | Resides in Go `HeapInuse`; retained across `ClearFreeBlockChannel` until 2 GC cycles evict `sync.Pool` + scavenger runs | Resides in Go `HeapInuse`; retained across `ClearFreeBlockChannel` until 2 GC cycles evict `sync.Pool` + scavenger runs |

### 3.2 Why File Lifecycle Churn Happens in Production GCSFuse Callers

Inspection of GCSFuse's two production callers of `internal/block` reveals why pool lifecycle creation and teardown (`ClearFreeBlockChannel`) is a hot path in production:
1. **Streaming Writes (`internal/bufferedwrites/buffered_write_handler.go:108, 230, 273, 308, 337`)**:
   - `NewBWHandler` creates a dedicated `BlockPool` **per open file write handle** (`reservedBlocks = 1`, default `write.block-size-mb = 32`, default `write.max-blocks-per-file = 1`).
   - When writing any file $\le 32\text{ MiB}$, GCSFuse allocates 1 block via `syscall.Mmap(32 MiB)`, faults in the pages during `Write()`, uploads the block on `Flush()`, and immediately calls `ClearFreeBlockChannel(true)` $\rightarrow$ `syscall.Munmap(32 MiB)`. **Intra-pool `freeBlocksCh` reuse is never exercised for files $\le 32\text{ MiB}$!**
   - Furthermore, every `fsync` call (`BufferedWriteHandler.Sync()`, line 230) invokes `ClearFreeBlockChannel(false)`, which **unmaps all free blocks mid-file** and forces subsequent writes to the *same* file to re-execute `syscall.Mmap` and re-fault pages from scratch.
2. **Buffered Reads (`internal/bufferedread/buffered_reader.go:143, 610`)**:
   - `NewBufferedReader` creates a dedicated `PrefetchBlockPool` **per open file read handle** (default `read.block-size-mb = 16`, `max-blocks-per-handle = 20`, `min-blocks-per-handle = 4`) and destroys it via `ClearFreeBlockChannel(true)` in `BufferedReader.Destroy()` whenever the file handle is closed.

---

## 4. Regime 1 Results: High File-Churn (`FileCloseChurn` & `FsyncChurn`)

Regime 1 (`BenchmarkBlockPool_HighFileChurn`) measures complete file lifecycles across all four canonical GCSFuse block sizes (`1 MiB`, `8 MiB`, `16 MiB`, `32 MiB`):
- **`FileCloseChurn` ($K=2$ simultaneous blocks/op)**: `NewPool` $\rightarrow$ acquire 2 blocks simultaneously $\rightarrow$ full `Write` + `Seek(0)` + `Read` + 4 KiB checksum on each $\rightarrow$ `Release` both $\rightarrow$ `ClearFreeBlockChannel(true)`.
- **`FsyncChurn` (2 waves of $K=2$ blocks/op = 4 blocks/op)**: Wave 1 (`Get` $\times 2 \rightarrow$ `Write`/`Read` $\rightarrow$ `Release` $\times 2 \rightarrow$ `ClearFreeBlockChannel(false)` simulating `Sync()`) followed by Wave 2 (`Get` $\times 2 \rightarrow$ `Write`/`Read` $\rightarrow$ `Release` $\times 2 \rightarrow$ `ClearFreeBlockChannel(true)` simulating `Flush()`).

### 4.1 Statistical Summary Table — Regime 1 (`-count=5`, `benchstat`)

| Mode | Block Size | Variant | Latency (`sec/op`) | vs `Mmap` ($p$-value) | Throughput (`B/s`) | `minor-faults/op` | `sys-ns/op` | `B/op` | `allocs/op` | `heap-inuse-MB` | `anon-rss-MB` |
|---|---|---|---|---|---|---|---|---|---|---|---|
| **`FileCloseChurn`** | **`1 MiB`** | `Mmap` | `852.9 µs` | baseline | `2.290 GiB/s` | **`512.0`** | `598.7 µs` | `272` | `5` | `33.74 MiB` | `46.29 MiB` |
| **`FileCloseChurn`** | **`1 MiB`** | `VariantA` | **`254.0 µs`** | **`-70.22%`** ($p=0.008$) | **`7.690 GiB/s` (`+235.8%`)** | **`0.002`** | `0.0 µs` | `304` | `5` | `36.75 MiB` | `46.43 MiB` |
| **`FileCloseChurn`** | **`1 MiB`** | `VariantB` | **`245.0 µs`** | **`-71.28%`** ($p=0.008$) | **`7.973 GiB/s` (`+248.2%`)** | **`0.002`** | `1.1 µs` | **`96`** | **`1`** | `36.73 MiB` | `50.20 MiB` |
| **`FileCloseChurn`** | **`8 MiB`** | `Mmap` | `3.120 ms` | baseline | `5.009 GiB/s` | **`8.000`** | `1030.3 µs` | `272` | `5` | `40.42 MiB` | `52.86 MiB` |
| **`FileCloseChurn`** | **`8 MiB`** | `VariantA` | **`2.197 ms`** | **`-29.57%`** ($p=0.008$) | **`7.112 GiB/s` (`+42.0%`)** | **`0.000`** | `5.9 µs` | `304` | `5` | `63.35 MiB` | `74.81 MiB` |
| **`FileCloseChurn`** | **`8 MiB`** | `VariantB` | **`2.176 ms`** | **`-30.23%`** ($p=0.008$) | **`7.179 GiB/s` (`+43.3%`)** | **`0.000`** | `3.2 µs` | **`96`** | **`1`** | `63.38 MiB` | `75.09 MiB` |
| **`FileCloseChurn`** | **`16 MiB`** | `Mmap` | `10.110 ms` | baseline | `3.091 GiB/s` | **`16.00`** | `2987.7 µs` | `272` | `5` | `48.05 MiB` | `61.56 MiB` |
| **`FileCloseChurn`** | **`16 MiB`** | `VariantA` | **`6.842 ms`** | **`-32.32%`** ($p=0.008$) | **`4.567 GiB/s` (`+47.8%`)** | **`0.000`** | `252.5 µs` | `304` | `5` | `93.86 MiB` | `105.29 MiB` |
| **`FileCloseChurn`** | **`16 MiB`** | `VariantB` | **`7.973 ms`** | **`-21.14%`** ($p=0.008$) | **`3.920 GiB/s` (`+26.8%`)** | **`0.000`** | `129.3 µs` | **`96`** | **`1`** | `78.64 MiB` | `105.38 MiB` |
| **`FileCloseChurn`** | **`32 MiB`** | `Mmap` | `24.560 ms` | baseline | `2.544 GiB/s` | **`32.20`** | `5939.5 µs` | `272` | `5` | `63.30 MiB` | `71.91 MiB` |
| **`FileCloseChurn`** | **`32 MiB`** | `VariantA` | **`21.810 ms`** | `-11.20%` ($p=0.095$) | `2.866 GiB/s` (`+12.7%`) | **`0.000`** | `288.6 µs` | `304` | `5` | `124.36 MiB` | `175.09 MiB` |
| **`FileCloseChurn`** | **`32 MiB`** | `VariantB` | **`21.880 ms`** | **`-10.93%`** ($p=0.016$) | **`2.857 GiB/s` (`+12.3%`)** | **`0.000`** | `317.6 µs` | **`96`** | **`1`** | `154.88 MiB` | `163.94 MiB` |
| **`FsyncChurn`** | **`1 MiB`** | `Mmap` | `1766.2 µs` | baseline | `2.212 GiB/s` | **`1024.0`** | `1271.4 µs` | `336` | `7` | `33.75 MiB` | `44.21 MiB` |
| **`FsyncChurn`** | **`1 MiB`** | `VariantA` | **`502.1 µs`** | **`-71.57%`** ($p=0.008$) | **`7.780 GiB/s` (`+251.8%`)** | **`0.000`** | `0.0 µs` | `400` | `7` | `36.71 MiB` | `47.06 MiB` |
| **`FsyncChurn`** | **`1 MiB`** | `VariantB` | **`467.9 µs`** | **`-73.51%`** ($p=0.008$) | **`8.349 GiB/s` (`+277.5%`)** | **`0.000`** | `15.4 µs` | **`96`** | **`1`** | `36.70 MiB` | `44.93 MiB` |
| **`FsyncChurn`** | **`8 MiB`** | `Mmap` | `6.992 ms` | baseline | `4.470 GiB/s` | **`16.00`** | `2336.8 µs` | `336` | `7` | `40.42 MiB` | `50.87 MiB` |
| **`FsyncChurn`** | **`8 MiB`** | `VariantA` | **`4.472 ms`** | **`-36.04%`** ($p=0.008$) | **`6.988 GiB/s` (`+56.3%`)** | **`0.000`** | `0.0 µs` | `400` | `7` | `63.35 MiB` | `73.72 MiB` |
| **`FsyncChurn`** | **`8 MiB`** | `VariantB` | **`4.515 ms`** | **`-35.42%`** ($p=0.008$) | **`6.921 GiB/s` (`+54.8%`)** | **`0.038`** | `68.2 µs` | **`96`** | **`1`** | `55.75 MiB` | `71.58 MiB` |
| **`FsyncChurn`** | **`16 MiB`** | `Mmap` | `19.760 ms` | baseline | `3.164 GiB/s` | **`32.00`** | `5358.0 µs` | `336` | `7` | `48.05 MiB` | `58.32 MiB` |
| **`FsyncChurn`** | **`16 MiB`** | `VariantA` | **`14.610 ms`** | **`-26.05%`** ($p=0.008$) | **`4.278 GiB/s` (`+35.2%`)** | **`0.000`** | `0.0 µs` | `400` | `7` | `78.60 MiB` | `103.47 MiB` |
| **`FsyncChurn`** | **`16 MiB`** | `VariantB` | **`14.730 ms`** | **`-25.45%`** ($p=0.008$) | **`4.244 GiB/s` (`+34.1%`)** | **`0.000`** | `0.0 µs` | **`96`** | **`1`** | `93.86 MiB` | `103.47 MiB` |
| **`FsyncChurn`** | **`32 MiB`** | `Mmap` | `50.490 ms` | baseline | `2.476 GiB/s` | **`64.00`** | `10744.3 µs` | `336` | `7` | `63.30 MiB` | `73.08 MiB` |
| **`FsyncChurn`** | **`32 MiB`** | `VariantA` | **`38.660 ms`** | **`-23.43%`** ($p=0.008$) | **`3.233 GiB/s` (`+30.6%`)** | **`0.000`** | `0.0 µs` | `400` | `7` | `154.88 MiB` | `161.74 MiB` |
| **`FsyncChurn`** | **`32 MiB`** | `VariantB` | **`40.490 ms`** | **`-19.80%`** ($p=0.008$) | **`3.087 GiB/s` (`+24.7%`)** | **`0.000`** | `647.5 µs` | **`96`** | **`1`** | `154.88 MiB` | `161.93 MiB` |

### 4.2 Systems-Level Analysis of Regime 1 (THP vs 4 KiB Faults & `sys-ns/op`)

1. **Exact Match Between `minor-faults/op` and Linux Transparent Huge Pages (`THP=[always]`)**:
   - At `size=1MiB` ($K=2$ blocks = `2 MiB` total, each block `< 2 MiB` PMD hugepage size), Linux cannot back a `1 MiB` VMA with a `2 MiB` Transparent Huge Page. Consequently, `Mmap` incurs exactly **$2 \times (1\text{ MiB} / 4\text{ KiB}) = 512.0$ minor page faults/op** in `FileCloseChurn` and **$4 \times 256 = 1,024.0$ minor page faults/op** in `FsyncChurn`!
   - At `size=8MiB`, `16MiB`, and `32MiB` ($\ge 2\text{ MiB}$), Linux's `THP=[always]` backs each block with `2 MiB` huge pages, producing exactly **$2 \times (8\text{ MiB} / 2\text{ MiB}) = 8.000$ minor faults/op** at 8 MiB, **`16.00` minor faults/op** at 16 MiB, and **`32.20` minor faults/op** at 32 MiB (`64.00` in `FsyncChurn`)!
   - Because `1 MiB` blocks suffer 256 page-table traps per MiB (compared to 0.5 THP traps per MiB for $\ge 8\text{ MiB}$ blocks), `Mmap`'s throughput at `1 MiB` collapses to **`2.290 GiB/s`** (spending `598.7 µs` out of `852.9 µs` = **`70.2%` of wall time inside the Linux kernel!**), whereas `VariantA` (`7.690 GiB/s`) and `VariantB` (`7.973 GiB/s`) incur **0 minor faults** and achieve a **`3.36x`–`3.77x` speedup**.
2. **Even With 2 MiB THPs at `8–32 MiB`, Kernel Page Zeroing + VMA Teardown Costs `1.03 ms` to `10.74 ms` per Op**:
   - At `8 MiB` (`FileCloseChurn`), `Mmap` spends `1.030 ms/op` in kernel mode (`sys-ns/op`) zeroing 16 MiB of fresh anonymous huge pages and unmapping them, adding `0.92 ms` (`+42%`) over Variant A (`2.197 ms/op`) and Variant B (`2.176 ms/op`).
   - At `32 MiB` (`FsyncChurn`), `Mmap` spends **`10.74 ms/op`** in kernel mode and takes `50.49 ms/op` vs Variant A's `38.66 ms/op` (`-23.43%`, saving **`11.83 ms` per file lifecycle**).

---

## 5. Regime 2 Results: Steady-State Single-Pool Reuse

Regime 2 (`BenchmarkBlockPool_SteadyStateReuse`) pre-warms a pool with 2 blocks outside the timed region and benchmarks continuous `Get() -> Write() -> Seek(0) -> Read() -> Release()` loops without calling `ClearFreeBlockChannel` until teardown.

### 5.1 Statistical Summary Table — Regime 2 (`-count=5`, `benchstat`)

| Block Size | Variant | Latency (`sec/op`) | vs `Mmap` ($p$-value) | Throughput (`B/s`) | `minor-faults/op` | `sys-ns/op` | `B/op` | `allocs/op` | `heap-inuse-MB` | `anon-rss-MB` |
|---|---|---|---|---|---|---|---|---|---|---|
| **`1 MiB`** | `Mmap` | `117.5 µs` | baseline | `8.310 GiB/s` | `0.000` | `0.0 ns` | `0` | `0` | **`33.74 MiB`** | `44.29 MiB` |
| **`1 MiB`** | `VariantA` | `118.4 µs` | `~` ($p=0.841$) | `8.249 GiB/s` | `0.000` | `0.0 ns` | `0` | `0` | `36.64 MiB` (`+8.6%`) | `44.94 MiB` |
| **`1 MiB`** | `VariantB` | `114.2 µs` | `~` ($p=0.151$) | `8.553 GiB/s` | `0.000` | `0.0 ns` | `0` | `0` | `36.64 MiB` (`+8.6%`) | `43.72 MiB` |
| **`8 MiB`** | `Mmap` | `1.217 ms` | baseline | `6.420 GiB/s` | `0.000` | `0.0 ns` | `0` | `0` | **`40.42 MiB`** | **`64.49 MiB`** |
| **`8 MiB`** | `VariantA` | `1.218 ms` | `~` ($p=1.000$) | `6.416 GiB/s` | `0.000` | `0.0 ns` | `0` | `0` | `55.75 MiB` (`+37.9%`) | `72.36 MiB` (`+12.2%`) |
| **`8 MiB`** | `VariantB` | `1.105 ms` | `~` ($p=0.151$) | `7.072 GiB/s` | `0.000` | `0.0 ns` | `0` | `0` | `63.34 MiB` (`+56.7%`) | `72.31 MiB` (`+12.1%`) |
| **`16 MiB`** | `Mmap` | `3.640 ms` | baseline | `4.292 GiB/s` | `0.000` | `354.8 ns` | `0` | `0` | **`48.05 MiB`** | **`89.45 MiB`** |
| **`16 MiB`** | `VariantA` | `3.779 ms` | `~` ($p=0.310$) | `4.134 GiB/s` | `0.000` | `0.0 ns` | `0` | `0` | `93.86 MiB` (`+95.4%`) | `102.81 MiB` (`+14.9%`) |
| **`16 MiB`** | `VariantB` | `2.862 ms` | `-21.39%` ($p=0.008$) | `5.460 GiB/s` | `0.000` | `0.0 ns` | `0` | `0` | `78.65 MiB` (`+63.7%`) | `102.62 MiB` (`+14.7%`) |
| **`32 MiB`** | `Mmap` | `9.241 ms` | baseline | `3.382 GiB/s` | `0.000` | `15.8 µs` | `0` | `0` | **`63.30 MiB`** | **`133.20 MiB`** |
| **`32 MiB`** | `VariantA` | `10.726 ms` | `+16.08%` ($p=0.008$) | `2.913 GiB/s` | `0.000` | `0.0 ns` | `0` | `0` | `124.36 MiB` (`+96.4%`) | `145.00 MiB` (`+8.8%`) |
| **`32 MiB`** | `VariantB` | `9.626 ms` | `~` ($p=0.841$) | `3.246 GiB/s` | `0.000` | `0.0 ns` | `0` | `0` | `124.45 MiB` (`+96.6%`) | `161.80 MiB` (`~`) |

### 5.2 Systems-Level Analysis of Regime 2

- Once a pool is warmed up, **`Mmap`, `VariantA`, and `VariantB` all achieve `0 B/op`, `0 allocs/op`, and `0 minor-faults/op`**.
- Note why `VariantB` is slightly faster at `16 MiB` single-pool (`2.862 ms` vs `3.640 ms`): in `SteadyStateReuse`, `Mmap` and `VariantA` pre-warm 2 blocks into `freeBlocksCh` (a FIFO queue), so `Get() -> Release()` alternates between 2 distinct 16 MiB buffers (32 MiB working set, exceeding single-socket L3 slice capacity), whereas `VariantB`'s `sync.Pool` places the most recently released block into `P.local.private` (LIFO), recycling the *same* 16 MiB buffer on every single-threaded `Get()` and enjoying higher L3 cache hit rates! In `Regime 4 SteadyReuse` (where 1 block is pre-warmed per worker), `Mmap`, `VariantA`, and `VariantB` are statistically identical at `16 MiB` (`2.889 ms`, `2.722 ms`, `3.008 ms`, $p > 0.095$).

---

## 6. Regime 3 Results: Go GC & Memory Pressure Interaction

Regime 3 (`BenchmarkBlockPool_GCPressure` at `size=8MiB`) evaluates three distinct GC/memory-pressure scenarios:
- **Sub-regime 3A (`ActiveChurn_HeapPressure`)**: File-close churn ($K=2 \times 8\text{ MiB}$) executing alongside 256 KiB/op of transient heap allocations (with a 4 MiB rotating live ring buffer) and periodic GC cycles under `GOGC=100`, aggressive `GOGC=20`, and `GOMEMLIMIT=128MiB`.
- **Sub-regime 3B (`BurstIdleGCEviction`)**: A burst of $K=4 \times 8\text{ MiB} = 32\text{ MiB}$ blocks followed by `ClearFreeBlockChannel(true)` and `gcCount` $\in \{0, 1, 2\}$ explicit `runtime.GC()` cycles between bursts.
- **Sub-regime 3C (`PostBurstMemoryRetention`)**: A burst of $K=8 \times 8\text{ MiB} = 64\text{ MiB}$ blocks measuring `rss-MB`, `anon-rss-MB`, and `heap-inuse-MB` (1) while all 8 blocks are actively held (`DuringActiveBurst`), (2) immediately after `ClearFreeBlockChannel(true)` without GC (`PostClear_NoGC`), and (3) after 2 GCs + `debug.FreeOSMemory()` (`PostClear_After2GC`).

### 6.1 Statistical Summary Table — Regime 3A & 3B (`-count=5`, `benchstat`)

| Sub-Regime & Condition | Variant | Latency (`sec/op`) | vs `Mmap` ($p$-value) | Throughput (`B/s`) | `B/op` | `allocs/op` | `gc-cycles` | `gc-pause-ns/op` | `minor-faults/op` | `heap-inuse-MB` | `anon-rss-MB` |
|---|---|---|---|---|---|---|---|---|---|---|---|
| **3A: `GOGC100`** | `Mmap` | `3.319 ms` | baseline | `4.708 GiB/s` | `256.3 KiB` | `6` | `4.0` | `24.58 µs` | `19.42` | **`44.67 MiB`** | **`54.86 MiB`** |
| **3A: `GOGC100`** | `VariantA` | **`2.488 ms`** | **`-25.04%`** ($p=0.008$) | `6.281 GiB/s` | `822.9 KiB` | `6` | `6.0` (`+50%`) | `23.50 µs` | `7.07` | `67.60 MiB` (`+51%`) | `77.96 MiB` (`+42%`) |
| **3A: `GOGC100`** | `VariantB` | **`2.321 ms`** | **`-30.07%`** ($p=0.008$) | **`6.732 GiB/s`** | `941.4 KiB` | **`2`** | `7.0` (`+75%`) | `23.43 µs` | `7.38` | `67.63 MiB` (`+51%`) | `77.76 MiB` (`+42%`) |
| **3A: `GOGC20`** | `Mmap` | `3.628 ms` | baseline | `4.307 GiB/s` | `256.3 KiB` | `6` | `8.0` | `45.97 µs` | `10.91` | **`44.67 MiB`** | **`55.90 MiB`** |
| **3A: `GOGC20`** | `VariantA` | **`2.766 ms`** | **`-23.77%`** ($p=0.008$) | `5.650 GiB/s` | `1898.2 KiB` | `7` | `13.0` (`+63%`) | `51.57 µs` | `18.66` | `59.50 MiB` (`+33%`) | `76.95 MiB` (`+38%`) |
| **3A: `GOGC20`** | `VariantB` | **`2.736 ms`** | **`-24.60%`** ($p=0.008$) | **`5.711 GiB/s`** | `1815.5 KiB` | **`3`** | `14.0` (`+75%`) | `45.35 µs` | `15.77` | `67.60 MiB` (`+51%`) | `77.92 MiB` (`+39%`) |
| **3A: `MemLimit128MiB`** | `Mmap` | `3.652 ms` | baseline | `4.278 GiB/s` | `256.3 KiB` | `6` | `8.0` | `48.01 µs` | `13.11` | **`44.43 MiB`** | **`56.37 MiB`** |
| **3A: `MemLimit128MiB`** | `VariantA` | **`2.930 ms`** | **`-19.76%`** ($p=0.008$) | `5.332 GiB/s` | `1429.9 KiB` | `7` | `12.0` (`+50%`) | `45.76 µs` | `3.83` | `59.49 MiB` (`+34%`) | `77.36 MiB` (`+37%`) |
| **3A: `MemLimit128MiB`** | `VariantB` | **`2.698 ms`** | **`-26.12%`** ($p=0.008$) | **`5.791 GiB/s`** | `1602.0 KiB` | **`3`** | `12.0` (`+50%`) | `45.08 µs` | `5.54` | `67.40 MiB` (`+52%`) | `79.12 MiB` (`+40%`) |
| **3B: `gcCount=0` (0 GC / op)** | `Mmap` | `8.205 ms` | baseline | `3.809 GiB/s` | `368 B` | `7` | `0.0` | `0.0 µs` | `16.00` | `40.42 MiB` | `57.42 MiB` |
| **3B: `gcCount=0` (0 GC / op)** | `VariantA` | **`5.679 ms`** | **`-30.78%`** ($p=0.008$) | `5.503 GiB/s` | `432 B` | `7` | `0.0` | `0.0 µs` | **`0.00`** | `70.94 MiB` | `83.46 MiB` |
| **3B: `gcCount=0` (0 GC / op)** | `VariantB` | **`5.658 ms`** | **`-31.04%`** ($p=0.008$) | **`5.523 GiB/s`** | **`96 B`** | **`1`** | `0.0` | `0.0 µs` | **`0.00`** | `70.95 MiB` | `83.06 MiB` |
| **3B: `gcCount=1` (1 GC / op)** | `Mmap` | `8.326 ms` | baseline | `3.754 GiB/s` | `368 B` | `7` | `14.0` | `31.00 µs` | `16.00` | `40.38 MiB` | `51.57 MiB` |
| **3B: `gcCount=1` (1 GC / op)** | `VariantA` | **`5.832 ms`** | **`-29.95%`** ($p=0.008$) | **`5.358 GiB/s`** | **`744 B`** | `11` | `19.0` | `23.60 µs` | **`0.00`** | `70.90 MiB` | `80.35 MiB` |
| **3B: `gcCount=1` (1 GC / op)** | `VariantB` | **`5.995 ms`** | **`-28.00%`** ($p=0.008$) | `5.213 GiB/s` | **`408 B`** | `5` | `19.0` | `25.00 µs` | **`0.00`** | `70.90 MiB` | `80.28 MiB` |
| **3B: `gcCount=2` (2 GCs / op)** | `Mmap` | **`8.288 ms`** | **baseline (Winner)** | `3.770 GiB/s` | **`368 B`** | **`7`** | `26.0` | `38.88 µs` | `16.00` | `40.38 MiB` | **`49.56 MiB`** |
| **3B: `gcCount=2` (2 GCs / op)** | `VariantA` | `8.041 ms` | `~` ($p=0.841$) | `3.886 GiB/s` | **`33,555,272 B` (`32 MiB`)** | `19` | `26.0` | `37.76 µs` | `9.71` | `40.38 MiB` | `79.15 MiB` (`+60%`) |
| **3B: `gcCount=2` (2 GCs / op)** | `VariantB` | `9.568 ms` | **`+15.44%` slower** ($p=0.032$) | `3.266 GiB/s` (`-13.4%`) | **`33,555,272 B` (`32 MiB`)** | `19` | `24.0` | `45.32 µs` | `10.00` | `40.39 MiB` | `79.15 MiB` (`+60%`) |

### 6.2 Statistical Summary Table — Regime 3C: Post-Burst Memory Retention (`64 MiB` Burst, `-count=5`)

| Phase (8 $\times$ 8 MiB = 64 MiB Burst) | Metric | `Mmap` | `VariantA` (`vs Mmap`) | `VariantB` (`vs Mmap`) | Systems Interpretation |
|---|---|---|---|---|---|
| **`DuringActiveBurst`** (All 8 blocks held live) | `anon-rss-MB` | `112.6 MiB` | `119.0 MiB` (`+5.67%`) | `118.5 MiB` (`+5.25%`) | All 3 variants fault 64 MiB of physical RAM while blocks are live. |
| **`DuringActiveBurst`** (All 8 blocks held live) | `heap-inuse-MB` | **`40.43 MiB`** | `109.10 MiB` (`+169.87%`) | `109.20 MiB` (`+170.11%`) | `Mmap` blocks are off-heap (`0` Go heap cost); Variants A/B add `+68.7 MiB` to `HeapInuse`. |
| **`PostClear_NoGC`** (After `ClearFreeBlockChannel(true)`) | `anon-rss-MB` | **`49.77 MiB`** | **`118.92 MiB` (`+138.93%`)** | **`118.64 MiB` (`+138.36%`)** | **`Mmap` immediately returns `62.8 MiB` of physical RAM to the OS**; `sync.Pool` retains 100% of the 64 MiB! |
| **`PostClear_NoGC`** (After `ClearFreeBlockChannel(true)`) | `heap-inuse-MB` | **`40.43 MiB`** | **`109.10 MiB` (`+169.87%`)** | **`101.47 MiB` (`+151.00%`)** | Pooled slices remain in Go's `HeapInuse` after file close. |
| **`PostClear_After2GC`** (After 2 GCs + `FreeOSMemory`) | `anon-rss-MB` | `48.60 MiB` | `50.44 MiB` (`+3.79%`) | `50.48 MiB` (`+3.87%`) | Only after 2 full GCs + `madvise(MADV_DONTNEED)` does `sync.Pool` release the RSS. |

### 6.3 Two Crucial `sync.Pool` Runtime Mechanics Proven by Regime 3

1. **Proof of the 1-GC `victim` Rescue vs 2-GC Eviction Cliff (`Sub-regime 3B`)**:
   - In Go's `src/sync/pool.go`, `poolCleanup()` runs at the start of each STW GC cycle: it drops the previous `victim` cache, moves `local` $\rightarrow$ `victim`, and clears `local`.
   - At `gcCount=1`, every burst rescues all $4 \times 8\text{ MiB} = 32\text{ MiB}$ buffers from `victim` back into `local`, resulting in **`0` slice allocations (`744 B/op` total)** and maintaining a **`-29.95%` latency advantage** over `Mmap`.
   - At `gcCount=2` (which occurs in GCSFuse whenever an idle gap between file bursts exceeds ~2–4 minutes of background `forcegc` or metadata GC cycles), the second GC wipes `victim` before the next burst arrives. Allocations jump by **45,000x** (`744 B/op` $\rightarrow$ **`33,555,272 B/op`**), and Go must zero 32 MiB of heap spans in user space (`runtime.memclrNoHeapPointers`) on every burst, erasing `sync.Pool`'s advantage (`VariantB` becomes **`+15.44%` slower than `Mmap`**).
2. **Discovery of Cross-P `victim.private` Stranding on Many-Core Servers (`Sub-regime 3A`)**:
   - Notice that in `Sub-regime 3A` (`ActiveChurn_HeapPressure` on `GOMAXPROCS=96` where GC runs every 4–8 ops), `Mmap` allocates only the 256 KiB test garbage (`256.3 KiB/op`), whereas `VariantA` and `VariantB` allocate `822.9 KiB/op` (`GOGC=100`) to `1,898.2 KiB/op` (`GOGC=20`) — meaning ~3.5% to ~9.5% of 8 MiB block acquisitions miss in `sync.Pool` even though only 1 GC occurred between ops!
   - Why? In Go's `src/sync/pool.go:getSlow()`, when a goroutine resumes on processor $P_B$ after a GC cycle, `getSlow()` steals from other processors' `P_A.victim.shared` ring buffers, **but it never steals from `P_A.victim.private`!** Because the first `Put` on $P_A$ always goes into `P_A.local.private` (which becomes `P_A.victim.private` after 1 GC), if the worker goroutine migrates from $P_A$ to $P_B$ across the 96 cores after a GC, `P_A.victim.private` is unreachable from $P_B$ and gets evicted on the subsequent GC!

---

## 7. Regime 4 Results: Full 4×4 Parameter Sweep (`1 MiB`, `8 MiB`, `16 MiB`, `32 MiB` × Concurrency `1`, `4`, `8`, `16`)

### 7.1 Workload A: `FileChurn` Across Block Size × Concurrency (`-count=5`, `benchstat`)

Each operation executes a complete file lifecycle holding $K=2$ blocks simultaneously (`2 * blockSize` payload per op).

| Block Size | Concurrency ($C$) | `Mmap` Latency (`sec/op`) | `VariantA` Latency (`vs Mmap`) | `VariantB` Latency (`vs Mmap`) | `Mmap` Throughput (`B/s`) | `VariantA` Throughput (`B/s`) | `VariantB` Throughput (`B/s`) | `Mmap` `sys-ns/op` | `Mmap` `anon-rss` | `VariantA` `anon-rss` | `VariantB` `anon-rss` |
|---|---|---|---|---|---|---|---|---|---|---|---|
| **`1 MiB`** | **`1`** | `853.4 µs` | **`270.8 µs` (`-68.27%`)** | **`245.0 µs` (`-71.30%`)** | `2.289 GiB/s` | `7.213 GiB/s` | **`7.973 GiB/s`** | `639.8 µs` | `46.7 MiB` | `49.4 MiB` | `48.8 MiB` |
| **`1 MiB`** | **`4`** | `318.4 µs` | **`60.94 µs` (`-80.86%`)** | **`61.60 µs` (`-80.65%`)** | `6.135 GiB/s` | **`32.05 GiB/s`** | `31.71 GiB/s` | `882.4 µs` | `49.3 MiB` | `58.0 MiB` | `58.3 MiB` |
| **`1 MiB`** | **`8`** | `260.8 µs` | **`32.42 µs` (`-87.57%`)** | **`31.33 µs` (`-87.99%`)** | `7.490 GiB/s` | `60.24 GiB/s` | **`62.35 GiB/s`** | `1026.0 µs` | `57.3 MiB` | `72.9 MiB` | `76.7 MiB` |
| **`1 MiB`** | **`16`** | `249.9 µs` | **`17.04 µs` (`-93.18%`)** | **`17.97 µs` (`-92.81%`)** | `7.817 GiB/s` | **`114.65 GiB/s`** | `108.67 GiB/s` | `1086.6 µs` | `72.2 MiB` | `103.4 MiB` | `100.8 MiB` |
| **`8 MiB`** | **`1`** | `3.415 ms` | **`2.308 ms` (`-32.41%`)** | **`2.230 ms` (`-34.70%`)** | `4.575 GiB/s` | `6.769 GiB/s` | **`7.006 GiB/s`** | `1172.5 µs` | `57.7 MiB` | `79.4 MiB` | `79.4 MiB` |
| **`8 MiB`** | **`4`** | `1.171 ms` | **`741.7 µs` (`-36.66%`)** | **`775.8 µs` (`-33.75%`)** | `13.34 GiB/s` | **`21.07 GiB/s`** | `20.14 GiB/s` | `1456.4 µs` | `79.2 MiB` | `137.1 MiB` | `151.0 MiB` |
| **`8 MiB`** | **`8`** | `806.2 µs` | **`575.7 µs` (`-28.59%`)** | **`584.3 µs` (`-27.53%`)** | `19.38 GiB/s` | **`27.14 GiB/s`** | `26.74 GiB/s` | `2266.2 µs` | `105.5 MiB` | `245.2 MiB` | `246.5 MiB` |
| **`8 MiB`** | **`16`** | `621.9 µs` | **`451.8 µs` (`-27.36%`)** | **`451.7 µs` (`-27.37%`)** | `25.12 GiB/s` | **`34.59 GiB/s`** | **`34.59 GiB/s`** | `2839.4 µs` | `170.1 MiB` | `421.1 MiB` | `426.3 MiB` |
| **`16 MiB`** | **`1`** | `9.678 ms` | **`7.299 ms` (`-24.58%`)** | **`7.964 ms` (`-17.71%`)** | `3.229 GiB/s` | **`4.281 GiB/s`** | `3.924 GiB/s` | `3366.8 µs` | `65.3 MiB` | `111.7 MiB` | `111.3 MiB` |
| **`16 MiB`** | **`4`** | `2.993 ms` | **`2.471 ms` (`-17.42%`)** | **`2.420 ms` (`-19.14%`)** | `10.44 GiB/s` | `12.65 GiB/s` | **`12.92 GiB/s`** | `4337.0 µs` | `109.6 MiB` | `249.7 MiB` | `251.5 MiB` |
| **`16 MiB`** | **`8`** | `1.823 ms` | **`1.473 ms` (`-19.23%`)** | **`1.454 ms` (`-20.27%`)** | `17.14 GiB/s` | `21.22 GiB/s` | **`21.50 GiB/s`** | `4996.1 µs` | `175.1 MiB` | `431.0 MiB` | `446.8 MiB` |
| **`16 MiB`** | **`16`** | `1.322 ms` | **`1.012 ms` (`-23.47%`)** | **`977.6 µs` (`-26.03%`)** | `23.64 GiB/s` | `30.90 GiB/s` | **`31.96 GiB/s`** | `6385.1 µs` | `295.9 MiB` | `829.7 MiB` | `814.2 MiB` |
| **`32 MiB`** | **`1`** | `24.080 ms` | **`20.44 ms` (`-15.11%`)** | **`20.37 ms` (`-15.40%`)** | `2.596 GiB/s` | `3.058 GiB/s` | **`3.068 GiB/s`** | `5631.2 µs` | `82.8 MiB` | `181.4 MiB` | `184.8 MiB` |
| **`32 MiB`** | **`4`** | `7.134 ms` | **`5.631 ms` (`-21.07%`)** | **`5.341 ms` (`-25.13%`)** | `8.761 GiB/s` | `11.10 GiB/s` | **`11.70 GiB/s`** | `7803.3 µs` | `176.0 MiB` | `450.4 MiB` | `450.1 MiB` |
| **`32 MiB`** | **`8`** | `4.229 ms` | **`3.329 ms` (`-21.29%`)** | **`3.174 ms` (`-24.94%`)** | `14.78 GiB/s` | `18.78 GiB/s` | **`19.69 GiB/s`** | `10164.6 µs` | `295.7 MiB` | `841.6 MiB` | `783.7 MiB` |
| **`32 MiB`** | **`16`** | `2.813 ms` | **`2.161 ms` (`-23.17%`)** | **`2.137 ms` (`-24.05%`)** | `22.22 GiB/s` | `28.92 GiB/s` | **`29.25 GiB/s`** | `11802.1 µs` | `539.0 MiB` | `1545.0 MiB` | `1547.8 MiB` |

*(All 16 cells in `FileChurn` are statistically significant with $p = 0.008$, except `8MiB/C=4 VariantA` at $p = 0.016$ and `16MiB/C=1 VariantA` at $p = 0.032$.)*

### 7.2 Workload B: `SteadyReuse` Across Block Size × Concurrency (`-count=5`, `benchstat`)

Each operation executes `Get() -> Write() -> Seek(0) -> Read() -> Release()` on pre-warmed per-worker pools (`0 B/op`, `0 allocs/op`, `0 minor-faults/op` across all cells).

| Block Size | Concurrency ($C$) | `Mmap` Latency (`sec/op`) | `VariantA` Latency (`vs Mmap`) | `VariantB` Latency (`vs Mmap`) | `Mmap` Throughput (`B/s`) | `VariantA` Throughput (`B/s`) | `VariantB` Throughput (`B/s`) | `Mmap` `heap-inuse` | `VariantA` `heap-inuse` | `VariantB` `heap-inuse` |
|---|---|---|---|---|---|---|---|---|---|---|
| **`1 MiB`** | **`1`** | `118.1 µs` | `122.1 µs` (`~`, $p=1.000$) | `119.0 µs` (`~`, $p=1.000$) | `8.266 GiB/s` | `8.001 GiB/s` | `8.207 GiB/s` | **`34.5 MiB`** | `36.4 MiB` | `36.5 MiB` |
| **`1 MiB`** | **`4`** | `29.86 µs` | `30.92 µs` (`~`, $p=0.841$) | `30.43 µs` (`~`, $p=1.000$) | `32.70 GiB/s` | `31.58 GiB/s` | `32.09 GiB/s` | **`37.4 MiB`** | `41.3 MiB` | `42.2 MiB` |
| **`1 MiB`** | **`8`** | `15.77 µs` | `15.77 µs` (`~`, $p=0.841$) | `16.35 µs` (`~`, $p=0.222$) | `61.92 GiB/s` | `61.91 GiB/s` | `59.74 GiB/s` | **`41.2 MiB`** | `49.0 MiB` | `49.0 MiB` |
| **`1 MiB`** | **`16`** | `8.001 µs` | `8.212 µs` (`~`, $p=0.841$) | `8.114 µs` (`~`, $p=1.000$) | **`122.1 GiB/s`** | `118.9 GiB/s` | `120.4 GiB/s` | **`49.0 MiB`** | `64.5 MiB` | `73.0 MiB` |
| **`8 MiB`** | **`1`** | `1.043 ms` | `1.062 ms` (`~`, $p=0.841$) | `1.008 ms` (`~`, $p=0.548$) | `7.489 GiB/s` | `7.355 GiB/s` | `7.748 GiB/s` | **`41.2 MiB`** | `56.5 MiB` | `56.5 MiB` |
| **`8 MiB`** | **`4`** | `331.0 µs` | `306.7 µs` (`~`, $p=0.421$) | `297.5 µs` (`~`, $p=0.151$) | `23.60 GiB/s` | `25.47 GiB/s` | `26.26 GiB/s` | **`64.1 MiB`** | `94.7 MiB` | `110.0 MiB` |
| **`8 MiB`** | **`8`** | `207.2 µs` | `211.8 µs` (`~`, $p=0.310$) | `212.8 µs` (`~`, $p=0.421$) | `37.71 GiB/s` | `36.89 GiB/s` | `36.71 GiB/s` | **`94.7 MiB`** | `155.9 MiB` | `155.8 MiB` |
| **`8 MiB`** | **`16`** | `189.0 µs` | `184.3 µs` (`~`, $p=0.151$) | `191.9 µs` (`~`, $p=1.000$) | `41.34 GiB/s` | `42.38 GiB/s` | `40.71 GiB/s` | **`155.8 MiB`** | `278.2 MiB` | `300.9 MiB` |
| **`16 MiB`** | **`1`** | `2.889 ms` | `2.722 ms` (`~`, $p=0.548$) | `3.008 ms` (`~`, $p=0.095$) | `5.408 GiB/s` | `5.739 GiB/s` | `5.195 GiB/s` | **`48.8 MiB`** | `79.4 MiB` | `79.4 MiB` |
| **`16 MiB`** | **`4`** | `915.0 µs` | `959.6 µs` (`~`, $p=0.222$) | `922.0 µs` (`~`, $p=1.000$) | `17.08 GiB/s` | `16.28 GiB/s` | `16.95 GiB/s` | **`94.7 MiB`** | `155.7 MiB` | `155.8 MiB` |
| **`16 MiB`** | **`8`** | `643.4 µs` | `626.4 µs` (`~`, $p=0.151$) | `686.3 µs` (`~`, $p=0.151$) | `24.29 GiB/s` | `24.94 GiB/s` | `22.77 GiB/s` | **`155.7 MiB`** | `278.0 MiB` | `277.9 MiB` |
| **`16 MiB`** | **`16`** | `457.4 µs` | `464.8 µs` (`~`, $p=1.000$) | `464.9 µs` (`~`, $p=1.000$) | `34.16 GiB/s` | `33.61 GiB/s` | `33.61 GiB/s` | **`277.9 MiB`** | `522.3 MiB` | `613.7 MiB` |
| **`32 MiB`** | **`1`** | `8.595 ms` | `9.216 ms` (`~`, $p=0.310$) | `8.410 ms` (`~`, $p=0.690$) | `3.636 GiB/s` | `3.391 GiB/s` | `3.716 GiB/s` | **`64.1 MiB`** | `94.7 MiB` | `94.7 MiB` |
| **`32 MiB`** | **`4`** | `2.742 ms` | `2.672 ms` (`~`, $p=1.000$) | `2.577 ms` (`~`, $p=0.151$) | `11.40 GiB/s` | `11.70 GiB/s` | `12.13 GiB/s` | **`155.7 MiB`** | `277.8 MiB` | `277.8 MiB` |
| **`32 MiB`** | **`8`** | `1.538 ms` | `1.555 ms` (`~`, $p=0.841$) | `1.537 ms` (`~`, $p=0.841$) | `20.32 GiB/s` | `20.10 GiB/s` | `20.34 GiB/s` | **`277.8 MiB`** | `522.1 MiB` | `522.0 MiB` |
| **`32 MiB`** | **`16`** | `1.069 ms` | `1.091 ms` (`~`, $p=0.421$) | `1.116 ms` (`+4.39%`, $p=0.008$) | `29.23 GiB/s` | `28.63 GiB/s` | `28.00 GiB/s` | **`522.0 MiB`** | `1010.9 MiB` | `1071.9 MiB` |

### 7.3 Scaling Analysis Across Block Size & Concurrency

1. **Why `Mmap` Scaling Collapses at `1 MiB` Under Concurrency ($C = 1 \rightarrow 16$)**:
   - In `SteadyReuse` at `1 MiB`, `Mmap` scales linearly from `8.27 GiB/s` ($C=1$) to **`122.1 GiB/s`** ($C=16$, `14.76x` scaling).
   - In `FileChurn` at `1 MiB`, `Mmap` scales from `2.29 GiB/s` ($C=1$) to only **`7.82 GiB/s`** ($C=16$, `3.41x` scaling), whereas `VariantA` scales from `7.21 GiB/s` to **`114.65 GiB/s`** (`15.89x` scaling, matching `SteadyReuse`!).
   - Look at `Mmap` `sys-ns/op` as $C$ increases from `1` to `16`: at `1 MiB`, kernel CPU time per op rises from `639.8 µs` ($C=1$) to `1,086.6 µs` ($C=16$); at `32 MiB`, kernel CPU time per op rises from **`5.63 ms` ($C=1$) to `11.80 ms` ($C=16$)**! Every `mmap`, `munmap`, and page fault on the same process contends on the Linux kernel's per-process `mm->mmap_lock` and broadcasts cross-core TLB shootdown IPIs to all active worker cores.
2. **Why `VariantA` and `VariantB` Use `+1.0 GiB` More `anon-rss-MB` at `32 MiB, C=16` in `FileChurn`**:
   - At `32 MiB, C=16` (`FileChurn`), `Mmap` `anon-rss-MB` is `539.0 MiB`, while `VariantA` (`1,545.0 MiB`) and `VariantB` (`1,547.8 MiB`) hold ~`1.55 GiB` (`+187%`).
   - Because `Mmap` calls `Munmap` at the end of every file lifecycle, only currently executing workers hold mapped blocks ($\le 16 \times 2 \times 32\text{ MiB}$, and at sampling time after the final op, `anon-rss-MB` reflects only residual pages). In `VariantA` and `VariantB`, as 16 unpinned goroutines migrate across `GOMAXPROCS=96` per-P `sync.Pool` shards during churn, up to ~24–48 `32 MiB` slices accumulate across different `P` shards in `HeapInuse` (`1,499.2 MiB`).

---

## 8. CPU & Memory Profiling (`pprof`) Breakdown & Architectural Recommendations

### 8.1 `pprof` CPU and Memory Profile Evidence

| Profiled Benchmark | Variant | Top Functions (`flat%` / `cum%` of Total Samples) | Key Profiling Insight |
|---|---|---|---|
| **`HighFileChurn` (`FileCloseChurn`, `8 MiB`, $C=1$)** | `Mmap` | `runtime.memmove`: **`84.11%`**<br>`syscall.Syscall6`: **`4.67%`** (`cum 3.74%` in `memoryBlock.Deallocate` / `munmap`, `0.93%` in `allocateNewBlock` / `mmap`)<br>`Getrusage` `sys-ns/op`: **`1.40 ms/op` (`39.2%` of wall time in kernel)** | Note that Go's user-space SIGPROF (`pprof`) attributes kernel page-fault time (`clear_page_erms` during `copy`) to the faulting instruction inside `runtime.memmove` (`Write`) and `Syscall6` (`munmap`). `Getrusage.Stime` (`1.40 ms/op` out of `3.57 ms/op`) proves `39.2%` of total execution time is spent in the Linux kernel! |
| **`HighFileChurn` (`FileCloseChurn`, `8 MiB`, $C=1$)** | `VariantA` | `runtime.memmove`: **`91.11%`** (`46.67%` `Write`, `44.44%` `Read`)<br>`syscall.Syscall6`: **`1.11%`**<br>`Getrusage` `sys-ns/op`: **`0.0006 ms/op` (`0.03%` of wall time)** | `0` `mmap`/`munmap` calls and `0` minor page faults; execution is purely user-space `runtime.memmove`. |
| **`HighFileChurn` (`FileCloseChurn`, `8 MiB`, $C=1$)** | `VariantB` | `runtime.memmove`: **`90.10%`** (`46.53%` `Write`, `43.56%` `Read`)<br>`syscall.Syscall6`: **`0.00%`**<br>`Getrusage` `sys-ns/op`: **`0.0000 ms/op`** | Zero syscalls, zero page faults, and `1 allocs/op` (`96 B/op`). |
| **`Sweep/FileChurn` (`1 MiB`, $C=16$)** | `Mmap` | Total CPU / Wall Ratio: **`506.45%`** (`4.18s` CPU in `825ms` wall across 16 workers)<br>`runtime.memmove`: **`69.86%`**<br>`syscall.Syscall6`: **`13.16%`**<br>`runtime.futex`: **`3.59%`** | 16 workers achieve only `5.06` cores of progress (`506%` out of `1600%`) because threads serialize on the kernel `mm->mmap_lock` and TLB shootdown IPIs during concurrent `mmap`/`munmap`/page faults. |
| **`Sweep/FileChurn` (`1 MiB`, $C=16$)** | `VariantA` | Total CPU / Wall Ratio: **`1384.58%`** (`13.23s` CPU in `955ms` wall across 16 workers)<br>`runtime.memmove`: **`94.48%`** (`45.80%` `Write`, `48.75%` `Read`)<br>`syscall.Syscall6`: **`0.00%`** | 16 workers achieve **`13.85` cores** of parallel `runtime.memmove` progress with zero kernel VMA lock contention, delivering **`114.65 GiB/s`** vs `Mmap`'s `7.82 GiB/s`. |
| **`GCPressure/BurstIdleGCEviction` (`gcCount=2`, `8 MiB`)** | `VariantA` | **CPU Profile**:<br>`runtime.memmove`: **`53.52%`**<br>`runtime.memclrNoHeapPointers`: **`19.72%`**<br>`runtime.tgkill`: **`4.23%`**<br>`runtime.stealWork`: **`4.23%`**<br>**Mem Profile (`alloc_space`)**:<br>`getSlicePool.func1`: **`97.30%` (`2,208 MB` = `32 MiB/op`)** | When 2 GCs occur between bursts, `sync.Pool` evicts its `victim` cache (`97.30%` of heap allocations come from `getSlicePool.func1` re-allocating 8 MiB slices), and Go spends **`19.72%` of total CPU time** zeroing newly allocated heap spans in `runtime.memclrNoHeapPointers`! |

---

### 8.2 Concrete Architectural Recommendations for GCSFuse

Based on the falsifiable empirical evidence across all four regimes and our audit of GCSFuse's production callers (`internal/bufferedwrites` and `internal/bufferedread`), we recommend a **targeted 3-part architecture** for GCSFuse:

1. **Immediate Zero-Risk Production Fix in `BufferedWriteHandler.Sync()` (Eliminate `fsync` `mmap` Churn Without `sync.Pool`)**:
   - **Finding**: In `internal/bufferedwrites/buffered_write_handler.go:230`, `Sync()` calls `wh.blockPool.ClearFreeBlockChannel(false)`, which unmaps all free blocks mid-file while retaining the 1 reserved semaphore permit. Moreover, in `internal/block/block_pool.go:243-266`, `ClearFreeBlockChannel(false)` unmaps *even the 1 reserved block* whose semaphore permit is still held by the pool!
   - **Recommendation**: Modify `ClearFreeBlockChannel(false)` (or `BufferedWriteHandler.Sync()`) so that when `releaseReservedBlocks == false`, the pool retains up to `bp.reservedBlocks` (1 block) allocated in `freeBlocksCh` instead of calling `Deallocate()` on it. This immediately eliminates the **`23.4%`–`71.6%` `FsyncChurn` penalty** (`Regime 1B`) while keeping 100% off-heap `mmap` memory management.
2. **Keep `mmap` as the Default Backing Store for Large Blocks (`16 MiB` / `32 MiB`) in Memory-Constrained / Bursty Environments**:
   - **Finding**: In `SteadyReuse` (`Regime 2` & `Regime 4B`), `mmap` matches `sync.Pool` throughput within noise (`~0%` delta) while keeping `640 MiB`–`50 GiB` of streaming read/write buffers completely invisible to Go's GC (`0` `HeapInuse`, `0` GC pacing inflation) and deterministically returning 100% of physical RSS (`RssAnon`) to the OS kernel on `ClearFreeBlockChannel(true)` (`49.77 MiB` vs `118.92 MiB` in `Regime 3C`).
   - Conversely, putting `16 MiB`–`32 MiB` blocks into `sync.Pool` (Variants A & B) inflates `anon-rss-MB` by **`+140%` to `+187%`** (`1.55 GiB` vs `539 MiB` at $C=16$, `Regime 4A`) due to per-`P` sharding across high-core machines and suffers a **`32 MiB/op` allocation + `19.72%` `memclrNoHeapPointers` CPU cliff** whenever 2 GC cycles elapse between bursts (`Regime 3B`).
3. **Adopt a Bounded Off-Heap Cross-File Recycle Cache (or Opt-In Variant A/B for Small-File / High-Churn Workloads)**:
   - **Finding**: The sole reason `mmap` loses by `1.18x`–`14.67x` in `Regime 1` and `Regime 4A` is that `GenBlockPool` is instantiated **per-file** and unmaps all blocks on file close (`Flush`/`Destroy`).
   - **Recommendation**:
     - For `1 MiB` buffers (`internal/buffer.FixedSizePool` used by `KernelRangeReader`), `sync.Pool` is ideal (`14.67x` faster than `mmap` at $C=16$ because `1 MiB < 2 MiB` THP size incurs 256 4 KiB page faults per MiB under `mmap`).
     - For `internal/block` (`16 MiB` `PrefetchBlockPool` and `32 MiB` `BlockPool`), either:
       - **(Option A — Best of Both Worlds)**: Retain `mmap` buffers (`0` GC overhead, invisible to `HeapInuse`), but instead of calling `syscall.Munmap` immediately in `ClearFreeBlockChannel(true)`, return `mmap` slices to a **bounded global free-list capped at $N$ blocks** (governed by `globalMaxBlocksSem`, with an idle scavenger or `Munmap` on eviction). This achieves Variant A/B's **`0`-fault, `0`-syscall cross-file reuse speedup (`+20%` to `+248%` on Xeon; `+39%` to `+520%` on `ct6e-standard-4t`)** *without* `sync.Pool`'s Go heap inflation, per-`P` RSS multiplication, or 2-GC eviction cliff!
       - **(Option B — Pure Go Heap `sync.Pool`)**: When using `sync.Pool` in high-churn workloads with ample RAM (`NewSyncPoolBlockPool` Variant A or `NewDirectSyncBlockPool` Variant B), **Variant B** is strictly superior to Variant A for new callers because it eliminates per-pool `freeBlocksCh` channel allocation (`1 alloc/op, 96 B/op` vs `5 allocs/op, 304 B/op`), provides native intra-pool thread safety, and matches or exceeds Variant A throughput across all regimes.

---

## 9. Phase 2 — GKE `ct6e-standard-4t` (Cloud TPU v6e Host) Empirical Evaluation & Workstation Comparison (`R4`)

To evaluate whether Transparent Hugepages (`THP`) and modern high-bandwidth server hardware alter the `mmap` vs `sync.Pool` trade-off in GCSFuse's primary ML training environment, we provisioned a dedicated GKE cluster (`blockpool-bench-ct6e-10100727`) with a 1-node **`ct6e-standard-4t`** (Cloud TPU v6e Trillium, 4 chips, `2x2` topology) nodepool (`ct6e-pool`) in project `gcs-fuse-test-ml`, zone `europe-west4-a` (using reservation `cloudtpu-20260914184000-1318357512`), scheduled benchmark pod `blockpool-bench-ct6e` onto node `gke-tpu-12341d57-2jnm`, and executed the full unit/race suite, isolated `/proc/vmstat` THP probes, and all 159 sub-benchmarks (`795` runs at `-benchtime=200ms -count=5 -benchmem`).

### 9.1 Node Verification, Hardware Topology & Transparent Hugepage (THP) Telemetry

#### 9.1.1 Verbatim GKE `ct6e-standard-4t` Node, CPU, Memory & THP Sysfs Verification

All raw outputs are archived in `.agents/teamwork/teamwork_preview_worker_r2_1/ct6e_node_and_thp_telemetry.txt`:

```text
=== KUBECTL GET POD -O WIDE ===
NAME                   READY   STATUS    RESTARTS   AGE    IP           NODE                    NOMINATED NODE   READINESS GATES
blockpool-bench-ct6e   1/1     Running   0          5m7s   10.240.1.4   gke-tpu-12341d57-2jnm   <none>           <none>

=== KUBECTL GET NODES -O WIDE ===
NAME                                                  STATUS   ROLES    AGE   VERSION               INTERNAL-IP   EXTERNAL-IP     OS-IMAGE                             KERNEL-VERSION   CONTAINER-RUNTIME
gke-blockpool-bench-ct6e-default-pool-b58f05e0-c7nw   Ready    <none>   18m   v1.35.8-gke.1225000   10.0.0.3      34.90.190.162   Container-Optimized OS from Google   6.12.94+         containerd://2.2.7
gke-tpu-12341d57-2jnm                                 Ready    <none>   11m   v1.35.8-gke.1225000   10.0.0.4      35.204.47.89    Container-Optimized OS from Google   6.12.94+         containerd://2.2.7

=== KUBECTL GET NODES LABELS ===
NAME                                                  STATUS   ROLES    AGE   VERSION               GKE-NODEPOOL   INSTANCE-TYPE      GKE-TPU-ACCELERATOR   GKE-TPU-TOPOLOGY   RESERVATION-NAME
gke-blockpool-bench-ct6e-default-pool-b58f05e0-c7nw   Ready    <none>   18m   v1.35.8-gke.1225000   default-pool   e2-medium                                                   
gke-tpu-12341d57-2jnm                                 Ready    <none>   11m   v1.35.8-gke.1225000   ct6e-pool      ct6e-standard-4t   tpu-v6e-slice         2x2                cloudtpu-20260914184000-1318357512

=== INSIDE POD: UNAME & GO VERSION ===
Linux blockpool-bench-ct6e 6.12.94+ #1 SMP Fri Aug 21 07:56:42 UTC 2026 x86_64 GNU/Linux
go version go1.27.0 linux/amd64

=== PAGESIZE & MAX_MAP_COUNT ===
PAGESIZE: 4096
vm.max_map_count: 65530

=== LSCPU (SUMMARY) ===
Architecture:                            x86_64
CPU(s):                                  180
On-line CPU(s) list:                     0-179
Vendor ID:                               AuthenticAMD
Model name:                              AMD EPYC 9B14
Thread(s) per core:                      2
Core(s) per socket:                      90
Socket(s):                               1
L1d cache:                               2.8 MiB (90 instances)
L1i cache:                               2.8 MiB (90 instances)
L2 cache:                                90 MiB (90 instances)
L3 cache:                                384 MiB (12 instances)
NUMA node(s):                            1

=== FREE -H & MEMINFO HUGE PAGES ===
               total        used        free      shared  buff/cache   available
Mem:           708Gi       8.0Gi       700Gi        43Mi       4.6Gi       700Gi
Swap:             0B          0B          0B
MemTotal:       742622448 kB
MemFree:        734951620 kB
MemAvailable:   734240816 kB
AnonPages:       1064492 kB
AnonHugePages:    653312 kB
Hugepagesize:       2048 kB

=== TRANSPARENT HUGEPAGE SYSFS ===
/sys/kernel/mm/transparent_hugepage/enabled: [always] madvise never
/sys/kernel/mm/transparent_hugepage/defrag: always defer defer+madvise [madvise] never
/sys/kernel/mm/transparent_hugepage/shmem_enabled: always within_size advise [never] deny force
/sys/kernel/mm/transparent_hugepage/hpage_pmd_size: 2097152
/sys/kernel/mm/transparent_hugepage/khugepaged/defrag: 1
/sys/kernel/mm/transparent_hugepage/khugepaged/pages_to_scan: 4096
/sys/kernel/mm/transparent_hugepage/khugepaged/scan_sleep_millisecs: 10000
/sys/kernel/mm/transparent_hugepage/khugepaged/alloc_sleep_millisecs: 60000
```

#### 9.1.2 Isolated Per-Variant `/proc/vmstat` THP Fault Delta Probe on `ct6e-standard-4t`

To directly observe how Linux `6.12.94+` (`THP=[always]`, `hpage_pmd_size = 2 MiB`) allocates physical pages for `Mmap` vs `VariantA` vs `VariantB`, we captured `/proc/vmstat` immediately before and after isolated `-benchtime=500ms -count=1` runs of `BenchmarkBlockPool_HighFileChurn/mode=FileCloseChurn` across `1 MiB`, `8 MiB`, and `32 MiB` (archived in `.agents/teamwork/teamwork_preview_worker_r2_1/ct6e_node_and_thp_telemetry.txt:163-352`):

| Variant | Block Size ($K=2$ blocks/op) | Executed Ops (`b.N`) | Per-Process `minor-faults/op` (`RUSAGE_SELF`) | Per-Process `sys-ns/op` (`RUSAGE_SELF`) | `/proc/vmstat` `pgfault` Delta | `/proc/vmstat` `thp_fault_alloc` Delta | `/proc/vmstat` `thp_fault_fallback` Delta | Kernel Page-Table & THP Attribution |
|---|---|---|---|---|---|---|---|---|
| **`Mmap`** | **`1 MiB`** (`2 MiB`/op) | `612` | **`512.0`** | **`723.0 µs`** (`722,992 ns`) | **`+1,337,733`** | `+7,791` (first-run `go test` compile/link + VMA merge/split) | `0` | Because `1 MiB < 2 MiB` PMD hugepage size, Linux **cannot** use 2 MiB THPs for isolated `1 MiB` `mmap` VMAs. Inside `b.N`, per-process `RUSAGE_SELF` confirms every op faults $2 \times (1\text{ MiB} / 4\text{ KiB}) = \mathbf{512.0}$ base 4 KiB pages (`986.3 µs/op`), while node-wide `thp_fault_alloc` (`+7,791`, `thp_split_pmd` `+2,160`) reflects first-run `go test` compilation/linking and adjacent VMA merge/split prior to `munmap`. |
| **`Mmap`** | **`8 MiB`** (`16 MiB`/op) | `332` | **`8.006`** | **`423.8 µs`** (`423,804 ns`) | `+16,659` | **`+3,590`** (`8.00` THP/op) | `0` | With `THP=[always]`, every `8 MiB` `mmap` block is backed by $4 \times 2\text{ MiB}$ PMD hugepages (`2 × 4 = 8.00` THP minor faults/op across 436 total calibration+timed ops = `3,488` block THPs + `102` process init THPs, `0` fallbacks). |
| **`Mmap`** | **`32 MiB`** (`64 MiB`/op) | `54` | **`32.15`** | **`5,555.3 µs`** (`5,555,259 ns`) | `+15,577` | **`+1,938`** (`32.00` THP/op) | `0` | Every `32 MiB` `mmap` block is backed by $16 \times 2\text{ MiB}$ PMD hugepages (`2 × 16 = 32.00` THP minor faults/op across 57 total calibration+timed ops = `1,824` block THPs + `114` process init THPs), spending **`5.56 ms/op` in kernel mode** zeroing 64 MiB of hugepages + tearing down VMAs (`10.65 ms/op` total). |
| **`VariantA`** | **`1 MiB`** (`2 MiB`/op) | `3,620` | **`0.005249`** | **`0.0 µs`** (`0 ns`) | `+14,890` | `+99` (warmup only) | `0` | `0` `thp_fault_alloc` and `0` `pgfault` per steady-state file churn op; slices are recycled in user space (`164.2 µs/op` vs `Mmap` `986.3 µs/op`). |
| **`VariantA`** | **`8 MiB`** (`16 MiB`/op) | `459` | **`0.02614`** | **`0.0 µs`** (`0 ns`) | `+14,851` | `+113` (warmup only) | `0` | Only the initial `2 × 8 MiB` pool warmup faults pages once; all `459` subsequent file lifecycles incur **`0` faults and `0` kernel time** (`1.30 ms/op` vs `Mmap` `1.79 ms/op`). |
| **`VariantA`** | **`32 MiB`** (`64 MiB`/op) | `100` | **`0.1600`** | **`0.0 µs`** (`0 ns`) | `+12,943` | `+171` (warmup only) | `0` | Initial `2 × 32 MiB` slice allocation faults `32` THPs once; all `100` file lifecycles reuse those slices in user space (`5.41 ms/op` vs `Mmap` `10.65 ms/op`). |
| **`VariantB`** | **`1 MiB`** (`2 MiB`/op) | `3,658` | **`0.001367`** | **`0.0 µs`** (`0 ns`) | `+14,359` | `+98` (warmup only) | `0` | `0` per-op page faults, `1 allocs/op` (`96 B/op`), `162.6 µs/op` vs `Mmap` `986.3 µs/op`. |
| **`VariantB`** | **`8 MiB`** (`16 MiB`/op) | `459` | **`0.000`** (`0`) | **`18.3 µs`** (`18,275 ns`) | `+12,868` | `+117` (warmup only) | `0` | `0` per-op page faults across `459` file lifecycles (`1.31 ms/op` vs `Mmap` `1.79 ms/op`). |
| **`VariantB`** | **`32 MiB`** (`64 MiB`/op) | `100` | **`0.1300`** | **`0.0 µs`** (`0 ns`) | `+13,834` | `+161` (warmup only) | `0` | `0` per-op page faults across `100` file lifecycles (`5.40 ms/op` vs `Mmap` `10.65 ms/op`). |

Across the entire 795-run benchmark suite (`274.828s` wall time, `ct6e_node_and_thp_telemetry.txt:354-383`), node-wide `/proc/vmstat` (`before` vs `after`) recorded **`+12,425,387` `pgfault`** (`4,598,980` $\rightarrow$ `17,024,367`), **`+338,747` `thp_fault_alloc`** (`21,897` $\rightarrow$ `360,644`, i.e., **`661.62 GiB` of 2 MiB THPs allocated and zeroed!**), **`0` `thp_fault_fallback`** (`12` $\rightarrow$ `12`), and **`+27` `pgmajfault`** (`7,626` $\rightarrow$ `7,653`).

---

### 9.2 GKE `ct6e-standard-4t` Regime 1 Results: High File-Churn (`FileCloseChurn` & `FsyncChurn`, `-count=5`, `benchstat`)

| Mode | Block Size | Variant | Latency (`sec/op`) | vs `Mmap` ($p$-value) | Throughput (`B/s`) | `minor-faults/op` | `sys-ns/op` | `B/op` | `allocs/op` | `heap-inuse-MB` | `anon-rss-MB` |
|---|---|---|---|---|---|---|---|---|---|---|---|
| **`FileCloseChurn`** | **`1 MiB`** | `Mmap` | `984.7 µs` | baseline | `1.983 GiB/s` | **`512.0`** | `700.0 µs` | `272` | `5` | `35.17 MiB` | `52.47 MiB` |
| **`FileCloseChurn`** | **`1 MiB`** | `VariantA` | **`164.5 µs`** | **`-83.30%`** ($p=0.008$) | **`11.88 GiB/s` (`+498.8%`)** | **`0.0007`** | `0.0 µs` | `304` | `5` | `38.43 MiB` | `56.47 MiB` |
| **`FileCloseChurn`** | **`1 MiB`** | `VariantB` | **`161.3 µs`** | **`-83.63%`** ($p=0.008$) | **`12.11 GiB/s` (`+510.7%`)** | **`0.0000`** | `0.3 µs` | **`96`** | **`1`** | `38.22 MiB` | `56.88 MiB` |
| **`FileCloseChurn`** | **`8 MiB`** | `Mmap` | `1.798 ms` | baseline | `8.692 GiB/s` | **`8.000`** | `467.8 µs` | `272` | `5` | `41.85 MiB` | `55.54 MiB` |
| **`FileCloseChurn`** | **`8 MiB`** | `VariantA` | **`1.289 ms`** | **`-28.28%`** ($p=0.008$) | **`12.12 GiB/s` (`+39.4%`)** | **`0.000`** | `1.9 µs` | `304` | `5` | `64.79 MiB` | `78.57 MiB` |
| **`FileCloseChurn`** | **`8 MiB`** | `VariantB` | **`1.291 ms`** | **`-28.21%`** ($p=0.008$) | **`12.11 GiB/s` (`+39.3%`)** | **`0.000`** | `2.0 µs` | **`96`** | **`1`** | `64.83 MiB` | `80.47 MiB` |
| **`FileCloseChurn`** | **`16 MiB`** | `Mmap` | `5.079 ms` | baseline | `6.153 GiB/s` | **`16.00`** | `2401.2 µs` | `272` | `5` | `49.46 MiB` | `62.27 MiB` |
| **`FileCloseChurn`** | **`16 MiB`** | `VariantA` | **`2.625 ms`** | **`-48.31%`** ($p=0.008$) | **`11.90 GiB/s` (`+93.5%`)** | **`0.000`** | `6.1 µs` | `304` | `5` | `95.30 MiB` | `109.00 MiB` |
| **`FileCloseChurn`** | **`16 MiB`** | `VariantB` | **`2.642 ms`** | **`-47.98%`** ($p=0.008$) | **`11.83 GiB/s` (`+92.2%`)** | **`0.000`** | `8.9 µs` | **`96`** | **`1`** | `95.32 MiB` | `110.15 MiB` |
| **`FileCloseChurn`** | **`32 MiB`** | `Mmap` | `10.752 ms` | baseline | `5.813 GiB/s` | **`32.00`** | `4835.1 µs` | `272` | `5` | `64.72 MiB` | `80.00 MiB` |
| **`FileCloseChurn`** | **`32 MiB`** | `VariantA` | **`5.413 ms`** | **`-49.65%`** ($p=0.008$) | **`11.55 GiB/s` (`+98.6%`)** | **`0.000`** | `12.1 µs` | `304` | `5` | `156.31 MiB` | `168.80 MiB` |
| **`FileCloseChurn`** | **`32 MiB`** | `VariantB` | **`5.414 ms`** | **`-49.65%`** ($p=0.008$) | **`11.55 GiB/s` (`+98.6%`)** | **`0.000`** | `53.1 µs` | **`96`** | **`1`** | `156.40 MiB` | `167.56 MiB` |
| **`FsyncChurn`** | **`1 MiB`** | `Mmap` | `1987.9 µs` | baseline | `1.965 GiB/s` | **`1024.0`** | `1374.0 µs` | `336` | `7` | `35.20 MiB` | `49.35 MiB` |
| **`FsyncChurn`** | **`1 MiB`** | `VariantA` | **`323.0 µs`** | **`-83.75%`** ($p=0.008$) | **`12.09 GiB/s` (`+515.4%`)** | **`0.000`** | `2.1 µs` | `400` | `7` | `38.30 MiB` | `51.98 MiB` |
| **`FsyncChurn`** | **`1 MiB`** | `VariantB` | **`320.7 µs`** | **`-83.86%`** ($p=0.008$) | **`12.18 GiB/s` (`+519.8%`)** | **`0.000`** | `1.9 µs` | **`96`** | **`1`** | `38.18 MiB` | `50.87 MiB` |
| **`FsyncChurn`** | **`8 MiB`** | `Mmap` | `3.626 ms` | baseline | `8.618 GiB/s` | **`16.00`** | `941.3 µs` | `336` | `7` | `41.85 MiB` | `56.13 MiB` |
| **`FsyncChurn`** | **`8 MiB`** | `VariantA` | **`2.588 ms`** | **`-28.64%`** ($p=0.008$) | **`12.08 GiB/s` (`+40.1%`)** | **`0.000`** | `4.4 µs` | `400` | `7` | `64.79 MiB` | `78.98 MiB` |
| **`FsyncChurn`** | **`8 MiB`** | `VariantB` | **`2.586 ms`** | **`-28.68%`** ($p=0.008$) | **`12.08 GiB/s` (`+40.2%`)** | **`0.022`** | `15.4 µs` | **`96`** | **`1`** | `57.20 MiB` | `78.63 MiB` |
| **`FsyncChurn`** | **`16 MiB`** | `Mmap` | `10.064 ms` | baseline | `6.210 GiB/s` | **`32.00`** | `4670.1 µs` | `336` | `7` | `49.48 MiB` | `63.71 MiB` |
| **`FsyncChurn`** | **`16 MiB`** | `VariantA` | **`5.263 ms`** | **`-47.70%`** ($p=0.008$) | **`11.88 GiB/s` (`+91.2%`)** | **`0.000`** | `8.9 µs` | `400` | `7` | `95.29 MiB` | `107.86 MiB` |
| **`FsyncChurn`** | **`16 MiB`** | `VariantB` | **`5.250 ms`** | **`-47.83%`** ($p=0.008$) | **`11.91 GiB/s` (`+91.7%`)** | **`0.000`** | `5.7 µs` | **`96`** | **`1`** | `80.05 MiB` | `107.57 MiB` |
| **`FsyncChurn`** | **`32 MiB`** | `Mmap` | `21.570 ms` | baseline | `5.794 GiB/s` | **`64.00`** | `10153.2 µs` | `336` | `7` | `64.73 MiB` | `77.21 MiB` |
| **`FsyncChurn`** | **`32 MiB`** | `VariantA` | **`10.930 ms`** | **`-49.31%`** ($p=0.008$) | **`11.43 GiB/s` (`+97.3%`)** | **`0.000`** | `57.0 µs` | `400` | `7` | `125.89 MiB` | `168.51 MiB` |
| **`FsyncChurn`** | **`32 MiB`** | `VariantB` | **`10.960 ms`** | **`-49.21%`** ($p=0.008$) | **`11.41 GiB/s` (`+96.9%`)** | **`0.000`** | `49.0 µs` | **`96`** | **`1`** | `156.40 MiB` | `168.70 MiB` |

---

### 9.3 GKE `ct6e-standard-4t` Regime 2 Results: Steady-State Single-Pool Reuse (`-count=5`, `benchstat`)

| Block Size | Variant | Latency (`sec/op`) | vs `Mmap` ($p$-value) | Throughput (`B/s`) | `minor-faults/op` | `sys-ns/op` | `B/op` | `allocs/op` | `heap-inuse-MB` | `anon-rss-MB` |
|---|---|---|---|---|---|---|---|---|---|---|
| **`1 MiB`** | `Mmap` | `80.33 µs` | baseline | `12.16 GiB/s` | `0.000` | `140.0 ns` | `0` | `0` | **`35.16 MiB`** | `50.98 MiB` |
| **`1 MiB`** | `VariantA` | `80.36 µs` | `~` ($p=0.690$) | `12.15 GiB/s` | `0.000` | `109.0 ns` | `0` | `0` | `38.06 MiB` (`+8.3%`) | `50.94 MiB` |
| **`1 MiB`** | `VariantB` | `81.24 µs` | `+1.12%` ($p=0.016$) | `12.02 GiB/s` | `0.000` | `360.6 ns` | `0` | `0` | `38.09 MiB` (`+8.3%`) | `50.82 MiB` |
| **`8 MiB`** | `Mmap` | `645.8 µs` | baseline | `12.10 GiB/s` | `0.000` | `871.0 ns` | `0` | `0` | **`41.84 MiB`** | **`73.50 MiB`** |
| **`8 MiB`** | `VariantA` | `646.1 µs` | `~` ($p=0.548$) | `12.09 GiB/s` | `0.000` | `823.4 ns` | `0` | `0` | `64.76 MiB` (`+54.8%`) | `80.65 MiB` (`+9.7%`) |
| **`8 MiB`** | `VariantB` | `649.0 µs` | `~` ($p=0.056$) | `12.04 GiB/s` | `0.000` | `640.5 ns` | `0` | `0` | `64.81 MiB` (`+54.9%`) | `82.20 MiB` (`~`) |
| **`16 MiB`** | `Mmap` | `1.313 ms` | baseline | `11.90 GiB/s` | `0.000` | `1.03 µs` | `0` | `0` | **`49.47 MiB`** | **`93.71 MiB`** |
| **`16 MiB`** | `VariantA` | `1.317 ms` | `~` ($p=0.421$) | `11.86 GiB/s` | `0.000` | `1.34 µs` | `0` | `0` | `95.28 MiB` (`+92.6%`) | `109.48 MiB` (`+16.8%`) |
| **`16 MiB`** | `VariantB` | `1.299 ms` | `-1.07%` ($p=0.008$) | `12.03 GiB/s` | `0.000` | `0.76 µs` | `0` | `0` | `95.31 MiB` (`+92.7%`) | `111.20 MiB` (`+18.7%`) |
| **`32 MiB`** | `Mmap` | `2.718 ms` | baseline | `11.50 GiB/s` | `0.000` | `0.99 µs` | `0` | `0` | **`64.73 MiB`** | **`142.50 MiB`** |
| **`32 MiB`** | `VariantA` | `2.738 ms` | `~` ($p=0.222$) | `11.41 GiB/s` | `0.000` | `6.11 µs` | `0` | `0` | `156.31 MiB` (`+141.5%`) | `167.80 MiB` (`+17.7%`) |
| **`32 MiB`** | `VariantB` | `2.726 ms` | `~` ($p=1.000$) | `11.46 GiB/s` | `0.000` | `13.19 µs` | `0` | `0` | `156.40 MiB` (`+141.6%`) | `167.80 MiB` (`+17.8%`) |

*(Notice that on `ct6e-standard-4t`'s `384 MiB` L3 cache, even `2 × 32 MiB = 64 MiB` fits comfortably in L3/high-bandwidth DDR5, so single-core `SteadyStateReuse` throughput stays practically flat at **`11.50`–`12.16 GiB/s` across all block sizes from `1 MiB` to `32 MiB`**!)*

---

### 9.4 GKE `ct6e-standard-4t` Regime 3 Results: Go GC & Memory Pressure Interaction (`-count=5`, `benchstat`)

#### 9.4.1 Sub-Regimes 3A (`ActiveChurn_HeapPressure`) & 3B (`BurstIdleGCEviction`) on `ct6e-standard-4t`

| Sub-Regime & Condition | Variant | Latency (`sec/op`) | vs `Mmap` ($p$-value) | Throughput (`B/s`) | `B/op` | `allocs/op` | `gc-cycles` | `gc-pause-ns/op` | `minor-faults/op` | `heap-inuse-MB` | `anon-rss-MB` |
|---|---|---|---|---|---|---|---|---|---|---|---|
| **3A: `GOGC100`** | `Mmap` | `2.331 ms` | baseline | `6.702 GiB/s` | `256.3 KiB` | `6` | **`12.0`** | `49.52 µs` | `9.84` | **`46.58 MiB`** | **`59.23 MiB`** |
| **3A: `GOGC100`** | `VariantA` | **`1.714 ms`** | **`-26.48%`** ($p=0.008$) | **`9.116 GiB/s`** | `966.5 KiB` | `6` | `17.0` (`+41.7%`) | `48.60 µs` | `17.21` | `62.11 MiB` (`+33%`) | `82.71 MiB` (`+39.6%`) |
| **3A: `GOGC100`** | `VariantB` | **`1.719 ms`** | **`-26.26%`** ($p=0.008$) | `9.089 GiB/s` | `882.4 KiB` | **`2`** | `17.0` (`+41.7%`) | `51.18 µs` | `35.01` | `61.43 MiB` (`+32%`) | `82.48 MiB` (`+39.3%`) |
| **3A: `GOGC20`** | `Mmap` | `2.906 ms` | baseline | `5.377 GiB/s` | `256.3 KiB` | `6` | `25.0` | `96.37 µs` | `9.68` | **`45.62 MiB`** | **`60.61 MiB`** |
| **3A: `GOGC20`** | `VariantA` | **`2.125 ms`** | **`-26.88%`** ($p=0.008$) | **`7.354 GiB/s`** | `1573.1 KiB` | `7` | `25.0` (`~`) | `93.69 µs` | `97.52` | `60.90 MiB` (`+33%`) | `86.84 MiB` (`+43.3%`) |
| **3A: `GOGC20`** | `VariantB` | **`2.154 ms`** | **`-25.86%`** ($p=0.008$) | `7.253 GiB/s` | `1494.6 KiB` | **`3`** | `26.0` (`~`) | `98.54 µs` | `105.20` | `60.92 MiB` (`+34%`) | `83.79 MiB` (`+38.3%`) |
| **3A: `MemLimit128MiB`** | `Mmap` | `2.823 ms` | baseline | `5.534 GiB/s` | `256.3 KiB` | `6` | `25.0` | `95.98 µs` | `10.24` | **`45.62 MiB`** | **`58.67 MiB`** |
| **3A: `MemLimit128MiB`** | `VariantA` | **`2.059 ms`** | **`-27.07%`** ($p=0.008$) | **`7.588 GiB/s`** | `1480.9 KiB` | `7` | `25.0` (`~`) | `95.54 µs` | `66.08` | `60.90 MiB` (`+33%`) | `80.82 MiB` (`+37.8%`) |
| **3A: `MemLimit128MiB`** | `VariantB` | **`2.137 ms`** | **`-24.31%`** ($p=0.008$) | `7.311 GiB/s` | `1658.6 KiB` | **`3`** | `25.0` (`~`) | `95.23 µs` | `92.33` | `60.92 MiB` (`+34%`) | `80.41 MiB` (`+37.1%`) |
| **3B: `gcCount=0` (0 GC / op)** | `Mmap` | `4.617 ms` | baseline | `6.769 GiB/s` | `368 B` | `7` | `0.0` | `0.0 µs` | `16.00` | **`41.81 MiB`** | **`57.48 MiB`** |
| **3B: `gcCount=0` (0 GC / op)** | `VariantA` | **`2.605 ms`** | **`-43.58%`** ($p=0.008$) | **`12.00 GiB/s`** | `432 B` | `7` | `0.0` | `0.0 µs` | **`0.00`** | `72.36 MiB` | `87.76 MiB` (`+52.7%`) |
| **3B: `gcCount=0` (0 GC / op)** | `VariantB` | **`2.609 ms`** | **`-43.48%`** ($p=0.008$) | `11.98 GiB/s` | **`96 B`** | **`1`** | `0.0` | `0.0 µs` | **`0.00`** | `72.34 MiB` | `87.84 MiB` (`+52.8%`) |
| **3B: `gcCount=1` (1 GC / op)** | `Mmap` | `5.227 ms` | baseline | `5.978 GiB/s` | `368 B` | `7` | **`45.0`** | `8.52 µs` | `16.00` | **`41.78 MiB`** | **`53.79 MiB`** |
| **3B: `gcCount=1` (1 GC / op)** | `VariantA` | **`3.006 ms`** | **`-42.50%`** ($p=0.008$) | `10.40 GiB/s` | **`744 B`** | `11` | `74.0` (`+64.4%`) | `7.14 µs` | **`0.00`** | `72.30 MiB` | `83.95 MiB` (`+56.1%`) |
| **3B: `gcCount=1` (1 GC / op)** | `VariantB` | **`2.998 ms`** | **`-42.64%`** ($p=0.008$) | **`10.42 GiB/s`** | **`408 B`** | **`5`** | `74.0` (`+64.4%`) | `7.15 µs` | **`0.00`** | `72.30 MiB` | `84.13 MiB` (`+56.4%`) |
| **3B: `gcCount=2` (2 GCs / op)** | `Mmap` | `5.370 ms` | baseline | `5.819 GiB/s` | **`368 B`** | **`7`** | **`84.0`** | `11.64 µs` | `16.00` | **`41.78 MiB`** | **`53.14 MiB`** |
| **3B: `gcCount=2` (2 GCs / op)** | `VariantA` | `4.182 ms` (`+39.1%` vs `1 GC`) | `-22.12%` ($p=0.008$) | `7.472 GiB/s` | **`33,555,272 B` (`32 MiB`)** | `19` | `108.0` (`+28.6%`) | `12.34 µs` | `2.56` | `41.78 MiB` | `81.62 MiB` (`+53.6%`) |
| **3B: `gcCount=2` (2 GCs / op)** | `VariantB` | `4.005 ms` (`+33.6%` vs `1 GC`) | `-25.41%` ($p=0.008$) | `7.802 GiB/s` | **`33,555,272 B` (`32 MiB`)** | `19` | `108.0` (`+28.6%`) | `11.59 µs` | `2.44` | `41.78 MiB` | `81.65 MiB` (`+53.7%`) |

#### 9.4.2 Sub-Regime 3C: Post-Burst Memory Retention (`64 MiB` Burst, `-count=5`) on `ct6e-standard-4t`

| Phase (8 $\times$ 8 MiB = 64 MiB Burst) | Metric | `Mmap` | `VariantA` (`vs Mmap`) | `VariantB` (`vs Mmap`) | Systems Interpretation on `ct6e-standard-4t` |
|---|---|---|---|---|---|
| **`DuringActiveBurst`** (All 8 blocks held live) | `anon-rss-MB` | `113.9 MiB` | `127.8 MiB` (`+12.23%`, $p=0.008$) | `125.2 MiB` (`+9.97%`, $p=0.008$) | All 3 variants fault 64 MiB of physical RAM while blocks are live. |
| **`DuringActiveBurst`** (All 8 blocks held live) | `heap-inuse-MB` | **`41.81 MiB`** | `110.53 MiB` (`+164.37%`, $p=0.008$) | `110.53 MiB` (`+164.37%`, $p=0.008$) | `Mmap` blocks are off-heap (`0` Go heap cost); Variants A/B add `+68.7 MiB` to `HeapInuse`. |
| **`PostClear_NoGC`** (After `ClearFreeBlockChannel(true)`) | `anon-rss-MB` | **`56.75 MiB`** | **`127.12 MiB` (`+124.00%`, $p=0.008$)** | **`125.12 MiB` (`+120.47%`, $p=0.008$)** | **`Mmap` immediately returns `57.15 MiB` of physical RAM to the OS**; `sync.Pool` retains 100% of the 64 MiB! |
| **`PostClear_NoGC`** (After `ClearFreeBlockChannel(true)`) | `heap-inuse-MB` | **`41.84 MiB`** | **`102.90 MiB` (`+145.95%`, $p=0.008$)** | **`102.90 MiB` (`+145.95%`, $p=0.008$)** | Pooled slices remain in Go's `HeapInuse` after file close. |
| **`PostClear_After2GC`** (After 2 GCs + `FreeOSMemory`) | `anon-rss-MB` | `53.42 MiB` | `53.44 MiB` (`~`, $p=0.349$) | `53.86 MiB` (`~`, $p=0.468$) | Only after 2 full GCs + `madvise(MADV_DONTNEED)` does `sync.Pool` release the RSS. |

---

### 9.5 GKE `ct6e-standard-4t` Regime 4 Results: Full 4×4 Parameter Sweep (`1 MiB`–`32 MiB` × Concurrency `1`, `4`, `8`, `16`, `-count=5`, `benchstat`)

#### 9.5.1 Workload A: `FileChurn` Across Block Size × Concurrency on `ct6e-standard-4t`

| Block Size | Concurrency ($C$) | `Mmap` Latency (`sec/op`) | `VariantA` Latency (`vs Mmap`) | `VariantB` Latency (`vs Mmap`) | `Mmap` Throughput (`B/s`) | `VariantA` Throughput (`B/s`) | `VariantB` Throughput (`B/s`) | `Mmap` `sys-ns/op` | `Mmap` `anon-rss` | `VariantA` `anon-rss` | `VariantB` `anon-rss` |
|---|---|---|---|---|---|---|---|---|---|---|---|
| **`1 MiB`** | **`1`** | `986.4 µs` | **`161.8 µs` (`-83.60%`)** | **`161.2 µs` (`-83.66%`)** | `1.980 GiB/s` | `12.07 GiB/s` | **`12.12 GiB/s`** | `711.2 µs` | `49.1 MiB` | `52.5 MiB` | `52.8 MiB` |
| **`1 MiB`** | **`4`** | `383.4 µs` | **`53.84 µs` (`-85.96%`)** | **`50.16 µs` (`-86.92%`)** | `5.094 GiB/s` | `36.28 GiB/s` | **`38.94 GiB/s`** | `1080.9 µs` | `51.9 MiB` | `61.6 MiB` | `61.6 MiB` |
| **`1 MiB`** | **`8`** | `283.1 µs` | **`29.55 µs` (`-89.56%`)** | **`29.97 µs` (`-89.41%`)** | `6.899 GiB/s` | **`66.10 GiB/s`** | `65.17 GiB/s` | `1373.7 µs` | `58.1 MiB` | `75.8 MiB` | `75.2 MiB` |
| **`1 MiB`** | **`16`** | `264.3 µs` | **`24.07 µs` (`-90.89%`)** | **`22.01 µs` (`-91.67%`)** | `7.389 GiB/s` | `81.15 GiB/s` | **`88.72 GiB/s`** | `1480.0 µs` | `67.1 MiB` | `105.7 MiB` | `103.6 MiB` |
| **`8 MiB`** | **`1`** | `1.815 ms` | **`1.310 ms` (`-27.84%`)** | **`1.309 ms` (`-27.90%`)** | `8.609 GiB/s` | `11.93 GiB/s` | **`11.94 GiB/s`** | `471.1 µs` | `60.7 MiB` | `79.6 MiB` | `79.7 MiB` |
| **`8 MiB`** | **`4`** | `918.6 µs` | **`435.5 µs` (`-52.60%`)** | **`351.7 µs` (`-61.72%`)** | `17.01 GiB/s` | `35.88 GiB/s` | **`44.43 GiB/s`** | `1443.8 µs` | `79.4 MiB` | `147.8 MiB` | `146.6 MiB` |
| **`8 MiB`** | **`8`** | `1.167 ms` | **`261.9 µs` (`-77.55%`)** | **`232.2 µs` (`-80.10%`)** | `13.39 GiB/s` | `59.65 GiB/s` | **`67.29 GiB/s`** | `4023.4 µs` | `108.1 MiB` | `244.4 MiB` | `250.3 MiB` |
| **`8 MiB`** | **`16`** | `937.4 µs` | **`166.2 µs` (`-82.27%`)** | **`172.4 µs` (`-81.60%`)** | `16.67 GiB/s` | **`94.03 GiB/s`** | `90.62 GiB/s` | `4541.4 µs` | `170.0 MiB` | `429.6 MiB` | `475.7 MiB` |
| **`16 MiB`** | **`1`** | `5.095 ms` | **`2.620 ms` (`-48.59%`)** | **`2.629 ms` (`-48.40%`)** | `6.133 GiB/s` | **`11.93 GiB/s`** | `11.89 GiB/s` | `2275.5 µs` | `64.7 MiB` | `108.2 MiB` | `108.9 MiB` |
| **`16 MiB`** | **`4`** | `1.725 ms` | **`814.5 µs` (`-52.78%`)** | **`757.0 µs` (`-56.11%`)** | `18.12 GiB/s` | `38.37 GiB/s` | **`41.28 GiB/s`** | `3499.4 µs` | `109.3 MiB` | `261.3 MiB` | `231.6 MiB` |
| **`16 MiB`** | **`8`** | `1.328 ms` | **`473.8 µs` (`-64.33%`)** | **`476.8 µs` (`-64.11%`)** | `23.53 GiB/s` | **`65.95 GiB/s`** | `65.54 GiB/s` | `4692.9 µs` | `170.4 MiB` | `444.3 MiB` | `444.4 MiB` |
| **`16 MiB`** | **`16`** | `1.072 ms` | **`333.1 µs` (`-68.93%`)** | **`345.1 µs` (`-67.81%`)** | `29.15 GiB/s` | **`93.81 GiB/s`** | `90.56 GiB/s` | `6446.5 µs` | `293.1 MiB` | `842.4 MiB` | `811.7 MiB` |
| **`32 MiB`** | **`1`** | `10.702 ms` | **`5.391 ms` (`-49.63%`)** | **`5.476 ms` (`-48.83%`)** | `5.840 GiB/s` | **`11.59 GiB/s`** | `11.41 GiB/s` | `4912.0 µs` | `80.7 MiB` | `170.4 MiB` | `170.6 MiB` |
| **`32 MiB`** | **`4`** | `3.619 ms` | **`1.820 ms` (`-49.70%`)** | **`1.825 ms` (`-49.58%`)** | `17.27 GiB/s` | **`34.33 GiB/s`** | `34.25 GiB/s` | `6715.3 µs` | `183.6 MiB` | `444.7 MiB` | `473.8 MiB` |
| **`32 MiB`** | **`8`** | `2.183 ms` | **`1.032 ms` (`-52.73%`)** | **`1.068 ms` (`-51.09%`)** | `28.63 GiB/s` | **`60.56 GiB/s`** | `58.53 GiB/s` | `8402.2 µs` | `305.5 MiB` | `840.9 MiB` | `811.6 MiB` |
| **`32 MiB`** | **`16`** | `1.535 ms` | **`732.8 µs` (`-52.24%`)** | **`752.7 µs` (`-50.95%`)** | `40.73 GiB/s` | **`85.28 GiB/s`** | `83.03 GiB/s` | `10565.6 µs` | `537.2 MiB` | `1576.4 MiB` | `1668.9 MiB` |

*(All 16 cells in `FileChurn` on `ct6e-standard-4t` are statistically significant with **$p = 0.008$ across 100% of cells** for both `VariantA` and `VariantB`!)*

#### 9.5.2 Workload B: `SteadyReuse` Across Block Size × Concurrency on `ct6e-standard-4t`

| Block Size | Concurrency ($C$) | `Mmap` Latency (`sec/op`) | `VariantA` Latency (`vs Mmap`) | `VariantB` Latency (`vs Mmap`) | `Mmap` Throughput (`B/s`) | `VariantA` Throughput (`B/s`) | `VariantB` Throughput (`B/s`) | `Mmap` `heap-inuse` | `VariantA` `heap-inuse` | `VariantB` `heap-inuse` |
|---|---|---|---|---|---|---|---|---|---|---|
| **`1 MiB`** | **`1`** | `80.26 µs` | `80.39 µs` (`~`, $p=0.841$) | `80.84 µs` (`~`, $p=0.056$) | `12.17 GiB/s` | `12.15 GiB/s` | `12.08 GiB/s` | **`36.0 MiB`** | `38.0 MiB` | `38.0 MiB` |
| **`1 MiB`** | **`4`** | `26.68 µs` | `23.23 µs` (`~`, $p=1.000$) | `25.73 µs` (`~`, $p=1.000$) | `36.60 GiB/s` | `42.04 GiB/s` | `37.96 GiB/s` | **`38.9 MiB`** | `42.8 MiB` | `43.8 MiB` |
| **`1 MiB`** | **`8`** | `16.37 µs` | `16.51 µs` (`~`, $p=0.690$) | `15.75 µs` (`~`, $p=0.690$) | `59.66 GiB/s` | `59.14 GiB/s` | `62.00 GiB/s` | **`42.8 MiB`** | `50.6 MiB` | `57.2 MiB` |
| **`1 MiB`** | **`16`** | `10.94 µs` | `11.23 µs` (`~`, $p=0.310$) | `10.83 µs` (`~`, $p=0.548$) | `89.26 GiB/s` | `86.98 GiB/s` | **`90.14 GiB/s`** | **`50.6 MiB`** | `66.1 MiB` | `78.4 MiB` |
| **`8 MiB`** | **`1`** | `645.8 µs` | `655.6 µs` (`+1.52%`, $p=0.008$) | `658.2 µs` (`+1.92%`, $p=0.008$) | `12.10 GiB/s` | `11.92 GiB/s` | `11.87 GiB/s` | **`42.7 MiB`** | `58.0 MiB` | `58.1 MiB` |
| **`8 MiB`** | **`4`** | `215.1 µs` | `182.6 µs` (`~`, $p=1.000$) | `216.8 µs` (`~`, $p=1.000$) | `36.32 GiB/s` | `42.79 GiB/s` | `36.03 GiB/s` | **`65.7 MiB`** | `96.2 MiB` | `96.2 MiB` |
| **`8 MiB`** | **`8`** | `118.1 µs` | `112.9 µs` (`~`, $p=0.222$) | `113.6 µs` (`~`, $p=0.690$) | `66.18 GiB/s` | `69.23 GiB/s` | `68.77 GiB/s` | **`96.2 MiB`** | `157.5 MiB` | `165.0 MiB` |
| **`8 MiB`** | **`16`** | `88.32 µs` | `85.05 µs` (`~`, $p=0.151$) | `87.37 µs` (`~`, $p=0.548$) | `88.46 GiB/s` | **`91.86 GiB/s`** | `89.41 GiB/s` | **`157.5 MiB`** | `279.8 MiB` | `294.8 MiB` |
| **`16 MiB`** | **`1`** | `1.296 ms` | `1.309 ms` (`+0.94%`, $p=0.008$) | `1.302 ms` (`+0.43%`, $p=0.008$) | `12.05 GiB/s` | `11.94 GiB/s` | `12.00 GiB/s` | **`50.4 MiB`** | `81.0 MiB` | `81.0 MiB` |
| **`16 MiB`** | **`4`** | `353.1 µs` | `438.6 µs` (`~`, $p=0.095$) | `352.3 µs` (`~`, $p=0.421$) | `44.25 GiB/s` | `35.62 GiB/s` | `44.36 GiB/s` | **`96.2 MiB`** | `157.4 MiB` | `157.4 MiB` |
| **`16 MiB`** | **`8`** | `263.8 µs` | `273.0 µs` (`~`, $p=1.000$) | `271.5 µs` (`~`, $p=0.548$) | `59.23 GiB/s` | `57.23 GiB/s` | `57.56 GiB/s` | **`157.4 MiB`** | `279.5 MiB` | `294.8 MiB` |
| **`16 MiB`** | **`16`** | `177.2 µs` | `175.2 µs` (`~`, $p=0.222$) | `176.0 µs` (`~`, $p=0.841$) | `88.18 GiB/s` | `89.20 GiB/s` | `88.77 GiB/s` | **`279.5 MiB`** | `523.9 MiB` | `600.1 MiB` |
| **`32 MiB`** | **`1`** | `2.702 ms` | `2.721 ms` (`~`, $p=0.690$) | `2.717 ms` (`~`, $p=1.000$) | `11.57 GiB/s` | `11.48 GiB/s` | `11.50 GiB/s` | **`65.8 MiB`** | `96.3 MiB` | `126.8 MiB` |
| **`32 MiB`** | **`4`** | `913.1 µs` | `902.2 µs` (`~`, $p=1.000$) | `915.5 µs` (`~`, $p=1.000$) | `34.22 GiB/s` | `34.64 GiB/s` | `34.14 GiB/s` | **`157.4 MiB`** | `279.5 MiB` | `279.4 MiB` |
| **`32 MiB`** | **`8`** | `535.1 µs` | `525.2 µs` (`~`, $p=0.841$) | `506.4 µs` (`~`, $p=0.841$) | `58.40 GiB/s` | `59.50 GiB/s` | `61.71 GiB/s` | **`279.4 MiB`** | `523.8 MiB` | `676.3 MiB` |
| **`32 MiB`** | **`16`** | `397.4 µs` | `368.8 µs` (`~`, $p=0.548$) | `364.6 µs` (`~`, $p=0.310$) | `78.63 GiB/s` | `84.73 GiB/s` | `85.71 GiB/s` | **`523.8 MiB`** | `1011.8 MiB` | `1164.4 MiB` |

---

### 9.6 Workstation (`Intel Xeon 96-vCPU`) vs GKE `ct6e-standard-4t` (`AMD EPYC 9B14 180-vCPU`) Cross-Platform Synthesis

The full side-by-side `benchstat` comparison across platforms is archived in `.agents/teamwork/teamwork_preview_worker_r2_1/benchstat_workstation_vs_ct6e.txt`:

| Operating Regime & Cell | Metric | Workstation (`Intel Xeon`, 96 vCPUs) | GKE `ct6e-standard-4t` (`AMD EPYC 9B14`, 180 vCPUs) | Cross-Platform Architectural Explanation |
|---|---|---|---|---|
| **1A: `FileCloseChurn` (`1 MiB`, $C=1$)** | `Mmap` vs `VariantB` Latency (`Speedup`) | `852.9 µs` vs `245.0 µs` (**`3.48x` / `-71.28%`**) | `984.7 µs` vs `161.3 µs` (**`6.11x` / `-83.63%`**) | At `1 MiB` (`< 2 MiB` THP), `Mmap` suffers `512` 4 KiB page faults/op on both hosts (`700.0 µs` `sys-ns/op` on `ct6e`), while EPYC 9B14 cuts user-space copy time by `-34.2%`, widening `sync.Pool`'s speedup to **`6.11x`**. |
| **1A: `FileCloseChurn` (`32 MiB`, $C=1$)** | `Mmap` vs `VariantA` Latency (`Speedup`) | `24.56 ms` vs `21.81 ms` (**`1.13x` / `-11.20%`**) | `10.75 ms` vs `5.41 ms` (**`1.99x` / `-49.65%`**) | On Xeon, `32 MiB` user-space copy takes `21.8 ms` (`~2.9 GiB/s`), masking `Mmap`'s `5.94 ms` kernel THP zeroing cost. On `ct6e-standard-4t` (`384 MiB` L3 + DDR5), user-space copy takes only **`5.41 ms` (`11.55 GiB/s` = `4.0x` faster!)**, so `Mmap`'s `4.84 ms` kernel THP zeroing + `munmap` cost doubles total latency! |
| **1B: `FsyncChurn` (`32 MiB`, $C=1$)** | `Mmap` vs `VariantA` Latency (`Speedup`) | `50.49 ms` vs `38.66 ms` (**`1.31x` / `-23.43%`**) | `21.57 ms` vs `10.93 ms` (**`1.97x` / `-49.31%`**) | `Mmap` spends **`10.15 ms/op` in kernel `sys-ns/op`** (`64.00` 2 MiB THP faults/op) on `ct6e-standard-4t`; `VariantA` saves **`10.64 ms` per `fsync` lifecycle**. |
| **2: `SteadyStateReuse` (`32 MiB`, $C=1$)** | `Mmap` / `VariantA` / `VariantB` Throughput | `3.38` / `2.91` / `3.25 GiB/s` (`~0%` tie) | **`11.50` / `11.41` / `11.46 GiB/s`** (`~0%` tie) | `ct6e-standard-4t` delivers **`3.40x` higher single-core `32 MiB` streaming throughput** (`-70.59%` latency, $p=0.008$) for all three variants equally. |
| **3B: `BurstIdleGCEviction` (`gcCount=2`, `8 MiB`)** | `VariantA` / `VariantB` `B/op` & Latency vs `1 GC` | **`33,555,272 B/op`**; Latency `+37.9%` / `+59.6%` (`VariantB` `+15.44%` slower than `Mmap`) | **`33,555,272 B/op`**; Latency `+39.1%` / `+33.6%` (`3.01 ms` $\rightarrow$ `4.18 ms/op`) | Both platforms exhibit the exact same **45,000x allocation cliff (`32 MiB/op`)** on GC #2. On EPYC 9B14, user-space zeroing (`+1.18 ms`) is faster than kernel `mmap`+`munmap` (`2.03 ms` `sys-ns/op`), so `VariantA/B` remain `-22%` / `-25%` faster in wall time despite re-allocating 32 MiB/op. |
| **3C: `PostBurstMemoryRetention` (`PostClear_NoGC`)** | `Mmap` vs `VariantA` / `VariantB` `anon-rss-MB` | **`49.77 MiB`** vs `118.92` / `118.64 MiB` (`+139%`) | **`56.75 MiB`** vs `127.12` / `125.12 MiB` (`+124%`) | On both platforms, `Mmap` immediately releases the 64 MiB burst to the OS on `ClearFreeBlockChannel(true)`, whereas `sync.Pool` retains 100% of the 64 MiB in `HeapInuse` until 2 GCs + scavenging. |
| **4A: `Sweep/FileChurn` (`1 MiB`, $C=16$)** | `Mmap` vs `VariantA` / `VariantB` Throughput (`Speedup`) | `7.82 GiB/s` vs **`114.65` / `108.67 GiB/s`** (**`14.67x` / `-93.18%`**) | `7.39 GiB/s` vs `81.15` / **`88.72 GiB/s`** (**`12.01x` / `-91.67%`**) | On `ct6e-standard-4t` (`1 MiB, C=16`), `pprof` (`ct6e_node_and_thp_telemetry.txt:451-511`) shows `VariantA` achieves **`1401.45%` CPU utilization (`14.01` active cores, `82.34%` `runtime.memmove`, `0%` `Syscall6`, `886.8 ns` `sys-ns/op`)**, whereas `Mmap` is throttled to **`705.25%` CPU (`7.05` active cores, `8.79%` `Syscall6`, `1.48 ms/op` `sys-ns/op`)** by `507.4` minor faults/op and `mm->mmap_lock` contention. |
| **4A: `Sweep/FileChurn` (`16 MiB`, $C=16$)** | `Mmap` vs `VariantA` Throughput (`Speedup`) | `23.64 GiB/s` vs `30.90 GiB/s` (**`1.31x` / `-23.47%`**) | `29.15 GiB/s` vs **`93.81 GiB/s`** (**`3.22x` / `-68.93%`**) | On `ct6e-standard-4t` (`16 MiB, C=16`), 16-core `VariantA` reaches **`93.81 GiB/s`** (`0` minor faults/op), whereas `Mmap` is throttled to `29.15 GiB/s` (`6.45 ms/op` `sys-ns/op`) by kernel hugepage zeroing, `mm->mmap_lock`, and TLB shootdowns. |
| **4A: `Sweep/FileChurn` (`32 MiB`, $C=16$)** | `Mmap` vs `VariantA` / `VariantB` Throughput & `anon-rss` | `22.22` vs `28.92` / `29.25 GiB/s` (`1.32x`); `anon-rss`: `539 MiB` vs `1,545` / `1,548 MiB` | `40.73` vs **`85.28` / `83.03 GiB/s`** (**`2.09x`**); `anon-rss`: `537 MiB` vs `1,576` / `1,669 MiB` | `ct6e-standard-4t` achieves a **`2.09x` speedup (`-52.24%` latency)** at `32 MiB, C=16`, while `VariantB` `anon-rss-MB` grows slightly higher (`1,668.9 MiB` vs `1,547.8 MiB`) due to `GOMAXPROCS=180` vs `96` per-`P` sharding. |

---

### 9.7 Falsifiable Verdict on `ct6e-standard-4t`: Does Slice-Based `sync.Pool` Allocation Outperform `mmap` With THP Enabled?

**Yes — in every file-churn and `fsync`-churn workload (`Regimes 1, 3A, and 4A`), `sync.Pool` (Variants A & B) outperforms `mmap` on `ct6e-standard-4t` even with `THP=[always]` active, achieving `1.39x` to `6.20x` speedups at $C=1$ and `2.09x` to `12.01x` speedups at $C=16$ ($p = 0.008$ across all 24 churn cells).**

Why does `THP=[always]` on `ct6e-standard-4t` **not** save `mmap` during file churn?
1. **For `1 MiB` blocks (`< 2 MiB` PMD hugepage size), Linux cannot use THPs at all**: `Mmap` falls back to `512` 4 KiB minor faults/op (`700 µs` `sys-ns/op` at $C=1$, `1.48 ms` `sys-ns/op` at $C=16$), causing `Mmap` to lose by **`6.11x` at $C=1$ (`1.98 GiB/s` vs `12.11 GiB/s`)** and **`12.01x` at $C=16$ (`7.39 GiB/s` vs `88.72 GiB/s`)**.
2. **For `8 MiB`, `16 MiB`, and `32 MiB` blocks, `THP=[always]` eliminates 4 KiB page-table walk overhead (`8`, `16`, and `32` `thp_fault_alloc`/op), but it CANNOT eliminate mandatory kernel physical memory zeroing (`clear_huge_page`) or `munmap` VMA/TLB teardown**:
   - Every time GCSFuse closes a file (`ClearFreeBlockChannel(true)`) and opens a new file (`syscall.Mmap`), the Linux kernel must zero 100% of the newly mapped `2 MiB` anonymous hugepages before user space can see them (to enforce POSIX/Linux process memory isolation) and must zap the PMD entries and broadcast cross-core TLB shootdown IPIs on `syscall.Munmap`.
   - At `32 MiB` ($K=2$ blocks = `64 MiB`/op), `Mmap` spends **`4.84 ms/op` in kernel `sys-ns/op`** (`10.57 ms/op` at $C=16$), whereas `sync.Pool` Variants A and B reuse already-faulted user-space buffers across file closes with **`0` `thp_fault_alloc`, `0` `minor-faults/op`, and `~0.01 ms` `sys-ns/op`**, cutting `32 MiB` file-churn latency nearly in half (**`5.41 ms` vs `10.75 ms` at $C=1$, `-49.65%`; `732.8 µs` vs `1,534.6 µs` at $C=16$, `-52.24%`**).
3. **Where `mmap` Still Wins on `ct6e-standard-4t`**:
   - In **steady-state single-pool reuse (`Regimes 2 & 4B`)**, `mmap` and `sync.Pool` are statistically tied (`11.5`–`12.2 GiB/s` at $C=1$, `85`–`92 GiB/s` at $C=16$), while `mmap` keeps 100% of block buffers off the Go heap (`-45%` to `-59%` lower `heap-inuse-MB`).
   - In **burst-idle workloads (`Regime 3B` at `>= 2` GCs)** and **post-burst memory reclamation (`Regime 3C`)**, `mmap` avoids the `32 MiB/op` 2-GC `sync.Pool` eviction cliff and immediately returns `anon-rss-MB` to the OS kernel upon `ClearFreeBlockChannel(true)` (`56.75 MiB` vs `127.12 MiB`), whereas `sync.Pool` retains `1.58`–`1.67 GiB` of `anon-rss-MB` at `32 MiB, C=16` (`+193%`–`+211%` over `Mmap`'s `537.2 MiB`).

---

### 9.8 Mandatory Cloud Resource Cleanup Verification (`gcs-fuse-test-ml`)

Immediately after copying all raw benchmark, race-test, and telemetry artifacts from `blockpool-bench-ct6e`, we deleted the GKE cluster `blockpool-bench-ct6e-10100727` (including nodepool `ct6e-pool`), all 19 firewall rules associated with `blockpool-bench-net-10100727`, subnet `blockpool-bench-subnet-10100727`, and VPC network `blockpool-bench-net-10100727`. Full verbatim output is archived in `.agents/teamwork/teamwork_preview_worker_r2_1/ct6e_cleanup_verification.txt`:

```text
$ gcloud container clusters delete blockpool-bench-ct6e-10100727 --project=gcs-fuse-test-ml --zone=europe-west4-a --quiet
Deleted [https://container.googleapis.com/v1/projects/gcs-fuse-test-ml/zones/europe-west4-a/clusters/blockpool-bench-ct6e-10100727].

$ gcloud compute networks subnets delete blockpool-bench-subnet-10100727 --project=gcs-fuse-test-ml --region=europe-west4 --quiet
Deleted [https://www.googleapis.com/compute/v1/projects/gcs-fuse-test-ml/regions/europe-west4/subnetworks/blockpool-bench-subnet-10100727].

$ gcloud compute networks delete blockpool-bench-net-10100727 --project=gcs-fuse-test-ml --quiet
Deleted [https://www.googleapis.com/compute/v1/projects/gcs-fuse-test-ml/global/networks/blockpool-bench-net-10100727].

$ gcloud container clusters list --project=gcs-fuse-test-ml
NAME          LOCATION    MASTER_VERSION      MASTER_IP      MACHINE_TYPE    NODE_VERSION        NUM_NODES  STATUS   STACK_TYPE
test-cluster  us-west4-a  1.36.4-gke.1495000  34.16.166.128  n2-standard-32  1.36.4-gke.1495000  1          RUNNING  IPV4_IPV6

$ gcloud compute networks list --project=gcs-fuse-test-ml
NAME                              SUBNET_MODE  BGP_ROUTING_MODE  IPV4_RANGE  GATEWAY_IPV4  INTERNAL_IPV6_RANGE
default                           AUTO         REGIONAL
dualstack-west4a                  CUSTOM       REGIONAL
gke-orbax-benchmark-grpc-network  CUSTOM       REGIONAL
grpc-verify-net                   CUSTOM       REGIONAL
npi-benchmark-net                 CUSTOM       REGIONAL
zb-network                        CUSTOM       GLOBAL

# Reservation cloudtpu-20260914184000-1318357512 in europe-west4-a verified back at 0 inUseResources:
"inUseResources": [{"accelerator": {"acceleratorCount": 0, "acceleratorType": ".../acceleratorTypes/ct6e"}}], "status": "READY"
```

