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

Run the script with the path to the JSON log file:

```bash
python3 tools/handle_visualizer/visualizer.py <path_to_log_file>
```

Options:
*   `--output <file>`, `-o <file>`: Specify the output image file (default: `read_pattern.png`).

## Example

```bash
python3 tools/handle_visualizer/visualizer.py my_gcsfuse_logs.json
```

This will generate `read_pattern.png` showing the read patterns. Sequential reads will appear as a diagonal line (or straight line if time is on X and offset on Y), while random reads will appear scattered.
