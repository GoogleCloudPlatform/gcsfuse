# Handle Visualizer

This tool visualizes file read patterns from GCSFuse logs. It plots the offset of `ReadFile` operations over time for each file handle.

## Prerequisites

*   Python 3
*   `matplotlib` library

To install `matplotlib`, run:

```bash
pip install matplotlib
```

## Usage

### Static Analysis

Run the script with the path to an existing JSON log file:

```bash
python3 tools/handle_visualizer/visualizer.py <path_to_log_file>
```

### Live Integration with GCSFuse

To monitor GCSFuse logs in real-time and generate the graph parallelly while GCSFuse runs:

1.  **Configure GCSFuse to log to a file:**

    ```bash
    # Example starting GCSFuse
    gcsfuse --log-file=/tmp/gcsfuse.log --debug_fuse ...
    ```

2.  **Run the visualizer in live mode:**

    In a separate terminal:

    ```bash
    python3 tools/handle_visualizer/visualizer.py --live /tmp/gcsfuse.log
    ```

    The tool will tail the log file, update the graph in real-time, and save the updated plot to `read_pattern.png` (or specified output) every second. If you have a display environment, it will also show a window.

### Piping logs (Unix style)

You can also pipe the output of GCSFuse directly to the visualizer using `-` as the filename:

```bash
gcsfuse --foreground --debug_fuse ... | python3 tools/handle_visualizer/visualizer.py --live -
```

## Options

*   `--output <file>`, `-o <file>`: Specify the output image file (default: `read_pattern.png`).
*   `--live`: Enable live monitoring mode.
