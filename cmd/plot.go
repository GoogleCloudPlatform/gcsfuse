// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// ExecutePlotHgrmCmd is the standalone entry point for 'gcs-bench plot-hgrm'.
var ExecutePlotHgrmCmd = func() {
	rootCmd := newPlotHgrmCmd()
	rootCmd.SetArgs(os.Args[1:])
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("plot-hgrm command failed: %v", err)
	}
}

// newPlotHgrmCmd builds the Cobra command for 'gcs-bench plot-hgrm'.
func newPlotHgrmCmd() *cobra.Command {
	var outputPath string
	var scale string

	cmd := &cobra.Command{
		Use:   "plot-hgrm <file.hgrm> [<file2.hgrm> ...]",
		Short: "Plot .hgrm latency files as a frequency distribution SVG",
		Long: `plot-hgrm reads one or more HDR histogram .hgrm files produced by
'gcs-bench bench' and generates a latency frequency distribution chart
as a self-contained SVG file.

The chart shows:
  - X axis: latency in milliseconds, log scale
  - Y axis: number of operations (count per HDR bucket)

With a logarithmic X axis, lognormal latency data appears as a symmetric
bell curve. Multiple files are overlaid on the same chart for comparison.

Output is written to latency_distribution.svg (or the path given by
--output).

Example:
  # Plot a single track
  gcs-bench plot-hgrm results/bench-20260329-184200-unet3d-read-total-latency.hgrm

  # Overlay TTFB and total latency from the same run
  gcs-bench plot-hgrm \
      results/bench-20260329-184200-unet3d-read-ttfb.hgrm \
      results/bench-20260329-184200-unet3d-read-total-latency.hgrm

  # Compare two runs
  gcs-bench plot-hgrm \
      run1/bench-20260329-184200-unet3d-read-total-latency.hgrm \
      run2/bench-20260329-185500-unet3d-read-total-latency.hgrm \
      --output comparison.svg`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if scale != "log" && scale != "linear" {
				return fmt.Errorf("--scale must be 'log' or 'linear', got %q", scale)
			}
			return runPlotHgrm(args, outputPath, scale)
		},
	}

	cmd.Flags().StringVar(&outputPath, "output", "latency_distribution.svg",
		"Path for the output SVG file")
	cmd.Flags().StringVar(&scale, "scale", "log",
		"X axis scale: 'log' (default) or 'linear'")
	return cmd
}

// ── data types ──────────────────────────────────────────────────────────────

// hgrmBucket is one HDR bucket: the upper bound (ms) and the per-bucket count.
type hgrmBucket struct {
	valueMs float64
	count   int64
}

// hgrmDataset is all buckets parsed from a single .hgrm file.
type hgrmDataset struct {
	label   string
	buckets []hgrmBucket
}

// ── parsing ─────────────────────────────────────────────────────────────────

// parseHgrmFile parses an HDR histogram percentile-distribution text file.
//
// The format (produced by PercentilesPrint with valueScale=1000.0) is:
//
//	Value  Percentile  TotalCount  1/(1-Percentile)
//
// TotalCount is cumulative. The function returns per-bucket counts by
// differencing consecutive rows.
func parseHgrmFile(path string) (hgrmDataset, error) {
	f, err := os.Open(path)
	if err != nil {
		return hgrmDataset{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var values []float64
	var cumCounts []int64

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		v, err1 := strconv.ParseFloat(fields[0], 64)
		c, err2 := strconv.ParseInt(fields[2], 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		values = append(values, v)
		cumCounts = append(cumCounts, c)
	}
	if err := scanner.Err(); err != nil {
		return hgrmDataset{}, fmt.Errorf("read %s: %w", path, err)
	}
	if len(values) == 0 {
		return hgrmDataset{}, fmt.Errorf("%s: no data rows found", path)
	}

	// Diff cumulative counts → per-bucket counts.
	buckets := make([]hgrmBucket, len(values))
	for i, v := range values {
		var cnt int64
		if i == 0 {
			cnt = cumCounts[0]
		} else {
			cnt = cumCounts[i] - cumCounts[i-1]
		}
		if cnt < 0 {
			cnt = 0
		}
		buckets[i] = hgrmBucket{valueMs: v, count: cnt}
	}

	return hgrmDataset{label: hgrmShortLabel(path), buckets: buckets}, nil
}

// hgrmShortLabel derives a readable legend label from a file path.
func hgrmShortLabel(path string) string {
	name := filepath.Base(path)
	name = strings.TrimSuffix(name, ".hgrm")
	// Strip leading bench-YYYYMMDD-HHMMSS- prefix if present.
	parts := strings.SplitN(name, "-", 4)
	if len(parts) == 4 && parts[0] == "bench" &&
		len(parts[1]) == 8 && len(parts[2]) == 6 {
		return parts[3]
	}
	return name
}

// ── SVG generation ───────────────────────────────────────────────────────────

// svgPalette is the colour set for overlaid datasets (fill, stroke).
var svgPalette = [][2]string{
	{"rgba(31,119,180,0.18)", "#1f77b4"},
	{"rgba(255,127,14,0.18)", "#ff7f0e"},
	{"rgba(44,160,44,0.18)", "#2ca02c"},
	{"rgba(214,39,40,0.18)", "#d62728"},
	{"rgba(148,103,189,0.18)", "#9467bd"},
	{"rgba(140,86,75,0.18)", "#8c564b"},
	{"rgba(227,119,194,0.18)", "#e377c2"},
	{"rgba(127,127,127,0.18)", "#7f7f7f"},
}

func runPlotHgrm(paths []string, outputPath string, scale string) error {
	var datasets []hgrmDataset
	for _, p := range paths {
		ds, err := parseHgrmFile(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping %s: %v\n", p, err)
			continue
		}
		datasets = append(datasets, ds)
	}
	if len(datasets) == 0 {
		return fmt.Errorf("no valid .hgrm files could be read")
	}

	svg, err := buildSVG(datasets, scale == "log")
	if err != nil {
		return err
	}

	if err := os.WriteFile(outputPath, []byte(svg), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outputPath, err)
	}
	fmt.Printf("Saved: %s\n", outputPath)
	return nil
}

// ── SVG layout constants ─────────────────────────────────────────────────────

const (
	svgWidth       = 960
	svgHeight      = 520
	svgMarginTop   = 40
	svgMarginBot   = 70
	svgMarginLeft  = 90
	svgMarginRight = 30
	svgPlotW       = svgWidth - svgMarginLeft - svgMarginRight
	svgPlotH       = svgHeight - svgMarginTop - svgMarginBot
	svgLegendX     = svgMarginLeft + svgPlotW - 10 // right-aligned
	svgLegendY     = svgMarginTop + 14
)

// buildSVG renders all datasets into an SVG string.
// logScale=true → logarithmic X axis; logScale=false → linear X axis.
func buildSVG(datasets []hgrmDataset, logScale bool) (string, error) {
	// ── determine axis ranges ────────────────────────────────────────────────
	var xMin, xMax float64 = math.MaxFloat64, 0
	var yMax int64

	for _, ds := range datasets {
		for _, b := range ds.buckets {
			if b.count == 0 || b.valueMs <= 0 {
				continue
			}
			if b.valueMs < xMin {
				xMin = b.valueMs
			}
			if b.valueMs > xMax {
				xMax = b.valueMs
			}
			if b.count > yMax {
				yMax = b.count
			}
		}
	}
	if xMin == math.MaxFloat64 || xMax == 0 {
		return "", fmt.Errorf("no plottable data (all counts are zero or values are non-positive)")
	}

	// ── X axis bounds and transforms ─────────────────────────────────────────
	var xAxisMin, xAxisMax float64
	var xPx func(float64) float64
	var xTicks []logTick

	if logScale {
		// Round to nearest decade boundary.
		logXMin := math.Floor(math.Log10(xMin))
		logXMax := math.Ceil(math.Log10(xMax))
		if logXMin == logXMax {
			logXMax = logXMin + 1
		}
		xAxisMin = math.Pow(10, logXMin)
		xAxisMax = math.Pow(10, logXMax)
		xPx = func(v float64) float64 {
			if v <= 0 {
				return float64(svgMarginLeft)
			}
			t := (math.Log10(v) - math.Log10(xAxisMin)) / (math.Log10(xAxisMax) - math.Log10(xAxisMin))
			return float64(svgMarginLeft) + t*float64(svgPlotW)
		}
		xTicks = logScaleTicks(xAxisMin, xAxisMax)
	} else {
		// Linear: start from 0, round max up to a nice value.
		xAxisMin = 0
		xAxisMax = niceLinearMax(xMax)
		xPx = func(v float64) float64 {
			t := v / xAxisMax
			return float64(svgMarginLeft) + t*float64(svgPlotW)
		}
		xTicks = linearScaleTicks(xAxisMax, 8)
	}

	// Y: round up to a nice number.
	yNice := niceYMax(yMax)

	// ── Y coordinate transform ───────────────────────────────────────────────
	yPx := func(c int64) float64 {
		t := float64(c) / float64(yNice)
		return float64(svgMarginTop+svgPlotH) - t*float64(svgPlotH)
	}
	yBase := yPx(0)

	// ── Y axis ticks (5 intervals) ────────────────────────────────────────────
	yTicks := yAxisTicks(yNice, 5)

	// ── build SVG ────────────────────────────────────────────────────────────
	var b strings.Builder

	b.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" font-family="sans-serif" font-size="12">`,
		svgWidth, svgHeight, svgWidth, svgHeight))
	b.WriteString("\n")

	// Background.
	b.WriteString(fmt.Sprintf(`  <rect width="%d" height="%d" fill="#fafafa" rx="4"/>`, svgWidth, svgHeight))
	b.WriteString("\n")

	// Clip path so histogram bars don't overflow the plot area.
	b.WriteString(fmt.Sprintf(`  <defs><clipPath id="plotArea"><rect x="%d" y="%d" width="%d" height="%d"/></clipPath></defs>`,
		svgMarginLeft, svgMarginTop, svgPlotW, svgPlotH))
	b.WriteString("\n")

	// Plot area background.
	b.WriteString(fmt.Sprintf(`  <rect x="%d" y="%d" width="%d" height="%d" fill="white" stroke="#ccc"/>`,
		svgMarginLeft, svgMarginTop, svgPlotW, svgPlotH))
	b.WriteString("\n")

	// ── Y grid lines + Y tick labels ─────────────────────────────────────────
	b.WriteString(`  <!-- Y grid lines -->`)
	b.WriteString("\n")
	for _, yt := range yTicks {
		py := yPx(yt)
		b.WriteString(fmt.Sprintf(`  <line x1="%d" y1="%.1f" x2="%d" y2="%.1f" stroke="#e0e0e0" stroke-width="1"/>`,
			svgMarginLeft, py, svgMarginLeft+svgPlotW, py))
		b.WriteString("\n")
		label := formatCount(yt)
		b.WriteString(fmt.Sprintf(`  <text x="%d" y="%.1f" text-anchor="end" dominant-baseline="middle" fill="#555">%s</text>`,
			svgMarginLeft-6, py, label))
		b.WriteString("\n")
	}

	// ── X grid lines + X tick labels ─────────────────────────────────────────
	b.WriteString(`  <!-- X grid lines and ticks -->`)
	b.WriteString("\n")
	for _, xt := range xTicks {
		px := xPx(xt.value)
		strokeColor := "#e0e0e0"
		strokeWidth := "1"
		if xt.major {
			strokeColor = "#ccc"
			strokeWidth = "1.5"
		}
		b.WriteString(fmt.Sprintf(`  <line x1="%.1f" y1="%d" x2="%.1f" y2="%d" stroke="%s" stroke-width="%s"/>`,
			px, svgMarginTop, px, svgMarginTop+svgPlotH, strokeColor, strokeWidth))
		b.WriteString("\n")
		if xt.label != "" {
			b.WriteString(fmt.Sprintf(`  <text x="%.1f" y="%d" text-anchor="middle" fill="#555">%s</text>`,
				px, svgMarginTop+svgPlotH+18, xt.label))
			b.WriteString("\n")
		}
	}

	// ── axis lines ────────────────────────────────────────────────────────────
	b.WriteString(fmt.Sprintf(`  <line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#333" stroke-width="1.5"/>`,
		svgMarginLeft, svgMarginTop, svgMarginLeft, svgMarginTop+svgPlotH))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#333" stroke-width="1.5"/>`,
		svgMarginLeft, svgMarginTop+svgPlotH, svgMarginLeft+svgPlotW, svgMarginTop+svgPlotH))
	b.WriteString("\n")

	// ── axis titles ───────────────────────────────────────────────────────────
	b.WriteString(fmt.Sprintf(`  <text x="%.1f" y="%d" text-anchor="middle" font-size="13" fill="#333">Latency (ms)</text>`,
		float64(svgMarginLeft)+float64(svgPlotW)/2, svgMarginTop+svgPlotH+50))
	b.WriteString("\n")
	// Y title (rotated).
	cx := float64(svgMarginLeft) - 65
	cy := float64(svgMarginTop) + float64(svgPlotH)/2
	b.WriteString(fmt.Sprintf(`  <text transform="translate(%.1f,%.1f) rotate(-90)" text-anchor="middle" font-size="13" fill="#333">Operation count</text>`,
		cx, cy))
	b.WriteString("\n")

	// ── plot title ────────────────────────────────────────────────────────────
	scaleLabel := "log scale"
	if !logScale {
		scaleLabel = "linear scale"
	}
	titleText := fmt.Sprintf("Latency frequency distribution (%s)", scaleLabel)
	b.WriteString(fmt.Sprintf(`  <text x="%.1f" y="%d" text-anchor="middle" font-size="15" font-weight="bold" fill="#222">%s</text>`,
		float64(svgMarginLeft)+float64(svgPlotW)/2, svgMarginTop-12, titleText))
	b.WriteString("\n")

	// ── datasets ─────────────────────────────────────────────────────────────
	b.WriteString(`  <g clip-path="url(#plotArea)">`)
	b.WriteString("\n")

	for i, ds := range datasets {
		palette := svgPalette[i%len(svgPalette)]
		fillColor := palette[0]
		strokeColor := palette[1]

		// Build step-histogram points. Each HDR bucket spans from the previous
		// value (lower bound) to ds.valueMs[i] (upper bound).
		//
		// Points for a closed filled polygon (bottom → step shape → bottom):
		//   start at (x(lowerBound[0]), yBase)
		//   rise to  (x(lowerBound[0]), y(count[0]))
		//   for each bucket i:
		//     horizontal to (x(upperBound[i]), y(count[i]))
		//     vertical to   (x(upperBound[i]), y(count[i+1]))   [or yBase at end]
		//   drop to (x(upperBound[last]), yBase)

		buckets := ds.buckets

		// Compute lower bounds: lower[i] = buckets[i-1].valueMs; lower[0] = first value / 2 (approx)
		lowers := make([]float64, len(buckets))
		for j := range buckets {
			if j == 0 {
				// Approximate lower bound for the first bucket.
				lowers[0] = buckets[0].valueMs / 2.0
				if lowers[0] < xAxisMin {
					lowers[0] = xAxisMin
				}
			} else {
				lowers[j] = buckets[j-1].valueMs
			}
		}

		// In linear mode, clamp lower bounds below zero to zero.
		if !logScale {
			for j := range lowers {
				if lowers[j] < 0 {
					lowers[j] = 0
				}
			}
		}

		// Build fill polygon points.
		var pts []string
		// Start at base of leftmost bucket.
		pts = append(pts, fmt.Sprintf("%.2f,%.2f", xPx(lowers[0]), yBase))
		for j, bk := range buckets {
			if bk.count == 0 && (j == 0 || buckets[j-1].count == 0) {
				// Skip fully empty spans — but still need to keep geometry contiguous.
			}
			pts = append(pts, fmt.Sprintf("%.2f,%.2f", xPx(lowers[j]), yPx(bk.count)))
			pts = append(pts, fmt.Sprintf("%.2f,%.2f", xPx(bk.valueMs), yPx(bk.count)))
		}
		// Drop to base at the right edge.
		pts = append(pts, fmt.Sprintf("%.2f,%.2f", xPx(buckets[len(buckets)-1].valueMs), yBase))

		pointStr := strings.Join(pts, " ")
		b.WriteString(fmt.Sprintf(`    <polygon points="%s" fill="%s"/>`, pointStr, fillColor))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf(`    <polyline points="%s" fill="none" stroke="%s" stroke-width="1.8" stroke-linejoin="miter"/>`,
			// Stroke only the top edge (drop the base closure).
			strings.Join(pts[1:len(pts)-1], " "), strokeColor))
		b.WriteString("\n")
	}

	b.WriteString(`  </g>`)
	b.WriteString("\n")

	// ── legend ────────────────────────────────────────────────────────────────
	if len(datasets) > 1 || true { // always show legend
		lineH := 20
		padX, padY := 10, 8
		maxLabelW := 0
		for _, ds := range datasets {
			if len(ds.label) > maxLabelW {
				maxLabelW = len(ds.label)
			}
		}
		legW := 18 + padX*2 + maxLabelW*7 // approximate char width
		legH := len(datasets)*lineH + padY*2
		legX := svgMarginLeft + svgPlotW - legW - 10
		legY := svgMarginTop + 10

		b.WriteString(fmt.Sprintf(`  <rect x="%d" y="%d" width="%d" height="%d" fill="white" fill-opacity="0.85" stroke="#ccc" rx="3"/>`,
			legX, legY, legW, legH))
		b.WriteString("\n")

		for i, ds := range datasets {
			palette := svgPalette[i%len(svgPalette)]
			strokeColor := palette[1]
			fillColor := palette[0]
			ry := legY + padY + i*lineH
			// Colour swatch.
			b.WriteString(fmt.Sprintf(`  <rect x="%d" y="%d" width="14" height="12" fill="%s" stroke="%s"/>`,
				legX+padX, ry, fillColor, strokeColor))
			b.WriteString("\n")
			b.WriteString(fmt.Sprintf(`  <text x="%d" y="%d" dominant-baseline="middle" fill="#333">%s</text>`,
				legX+padX+18, ry+6, svgEscape(ds.label)))
			b.WriteString("\n")
		}
	}

	b.WriteString("</svg>\n")
	return b.String(), nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

// logTick is one tick on the log X axis.
type logTick struct {
	value float64
	major bool   // true = decade boundary (label shown)
	label string // empty = minor tick (no label)
}

// logScaleTicks returns tick marks spanning [xMin, xMax] (both powers of 10).
// Major ticks at each decade; minor ticks at 2× and 5× within each decade.
func logScaleTicks(xMin, xMax float64) []logTick {
	var ticks []logTick
	decade := xMin
	for decade <= xMax*1.001 {
		ticks = append(ticks, logTick{value: decade, major: true, label: formatMs(decade)})
		// Minor ticks at 2× and 5×.
		for _, mult := range []float64{2, 5} {
			v := decade * mult
			if v < xMax*1.001 {
				ticks = append(ticks, logTick{value: v, major: false, label: ""})
			}
		}
		decade *= 10
	}
	return ticks
}

// formatMs formats a millisecond value as a readable tick label.
func formatMs(ms float64) string {
	if ms < 1 {
		return fmt.Sprintf("%.2g ms", ms)
	}
	if ms < 10 {
		return fmt.Sprintf("%.1g ms", ms)
	}
	if ms < 1000 {
		return fmt.Sprintf("%.0f ms", ms)
	}
	return fmt.Sprintf("%.0f s", ms/1000)
}

// linearScaleTicks returns n evenly-spaced ticks from 0 to xMax.
func linearScaleTicks(xMax float64, n int) []logTick {
	ticks := make([]logTick, n+1)
	for i := 0; i <= n; i++ {
		v := xMax * float64(i) / float64(n)
		ticks[i] = logTick{value: v, major: true, label: formatMs(v)}
	}
	return ticks
}

// niceLinearMax rounds v up to a nice number suitable for a linear axis max.
func niceLinearMax(v float64) float64 {
	mag := math.Pow(10, math.Floor(math.Log10(v)))
	f := v / mag
	var nice float64
	switch {
	case f <= 1:
		nice = 1
	case f <= 2:
		nice = 2
	case f <= 5:
		nice = 5
	default:
		nice = 10
	}
	return nice * mag
}

// yAxisTicks returns n+1 evenly-spaced Y axis tick values from 0 to yMax.
func yAxisTicks(yMax int64, n int) []int64 {
	ticks := make([]int64, n+1)
	for i := 0; i <= n; i++ {
		ticks[i] = int64(float64(yMax) * float64(i) / float64(n))
	}
	return ticks
}

// niceYMax rounds v up to a "nice" number for axis scaling.
func niceYMax(v int64) int64 {
	if v == 0 {
		return 1
	}
	mag := math.Pow(10, math.Floor(math.Log10(float64(v))))
	f := float64(v) / mag
	var nice float64
	switch {
	case f <= 1:
		nice = 1
	case f <= 2:
		nice = 2
	case f <= 5:
		nice = 5
	default:
		nice = 10
	}
	return int64(nice * mag)
}

// formatCount formats a count for axis labels (e.g. 1000→"1k", 1500000→"1.5M").
func formatCount(c int64) string {
	switch {
	case c == 0:
		return "0"
	case c >= 1_000_000:
		if c%1_000_000 == 0 {
			return fmt.Sprintf("%dM", c/1_000_000)
		}
		return fmt.Sprintf("%.1fM", float64(c)/1_000_000)
	case c >= 1_000:
		if c%1_000 == 0 {
			return fmt.Sprintf("%dk", c/1_000)
		}
		return fmt.Sprintf("%.1fk", float64(c)/1_000)
	default:
		return strconv.FormatInt(c, 10)
	}
}

// svgEscape escapes a string for safe embedding in SVG text.
func svgEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
