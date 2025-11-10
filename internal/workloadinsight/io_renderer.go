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

package workloadinsight

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

const (
	blockChar   = "█"
	emptyChar   = " "
	labelHeader = "[offset,len)"
)

// humanReadable formats a byte size into a compact string (KB, MB, GB).
func humanReadable(size uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.1fG", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.1fM", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.1fK", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%dB", size)
	}
}

// Range represents a byte range [Start, End).
type Range struct {
	Start uint64
	End   uint64
}

// Renderer renders I/O byte ranges as ASCII plots to visualize the access patterns.
type Renderer struct {
	plotWidth  int // number of columns used for plotting area
	labelWidth int // width of left label column (0 = auto)
	pad        int // spaces between label and plot
}

func NewRenderer() (*Renderer, error) {
	return NewRendererWithSettings(100, 20, 2)
}

// NewRendererWithSettings returns a Renderer with the specified settings.
// Returns an error if any setting is invalid (e.g. negative).
func NewRendererWithSettings(plotWidth, labelWidth, pad int) (*Renderer, error) {
	if labelWidth < len(labelHeader) {
		return nil, fmt.Errorf("labelWidth must be at least %d", len(labelHeader))
	}

	if pad < 0 {
		return nil, fmt.Errorf("plotWidth and pad must be non-negative")
	}

	if plotWidth < 1 {
		return nil, fmt.Errorf("plotWidth must be positive")
	}

	return &Renderer{
		plotWidth:  plotWidth,
		labelWidth: labelWidth,
		pad:        pad,
	}, nil
}

// Render renders the given ranges for a single file of the specified size
// and returns the ASCII representation as a string.
func (r *Renderer) Render(name string, size uint64, ranges []Range) (string, error) {
	var sb strings.Builder
	header, err := r.buildHeader(name, size, ranges)
	if err != nil {
		return "", err
	}
	sb.WriteString(header)

	for i := range ranges {
		line, err := r.buildRow(size, ranges[i])
		if err != nil {
			return "", err
		}
		sb.WriteString(line)
		if i < len(ranges)-1 {
			sb.WriteByte('\n')
		}
	}
	sb.WriteByte('\n')
	return sb.String(), nil
}

// buildStats builds statistics about the given ranges for a single file
// and returns them as a string.
func (r *Renderer) buildStats(ranges []Range) string {
	length := len(ranges)
	if length <= 0 {
		return ""
	}

	sizes := make([]uint64, length)
	sum := uint64(0)
	for i, rg := range ranges {
		sizes[i] = rg.End - rg.Start
		sum += sizes[i]
	}
	sort.Slice(sizes, func(i, j int) bool { return sizes[i] < sizes[j] })

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Total IOs: %d\n", length))
	sb.WriteString(fmt.Sprintf("IO Size Distributions: (Min: %s, Median: %s, Max: %s, Avg: %s)\n", humanReadable(sizes[0]), humanReadable(sizes[length/2]), humanReadable(sizes[length-1]), humanReadable(sum/uint64(length))))
	return sb.String()
}

// buildHeader composes the header (filename, tick marks, numeric labels)
// and returns it as a string to be prepended to the chart rows.
// E.g.:
//
// Name: demo.txt
//
//	0B        250B         500B        750B      1000B
//
// [offset,len)          |-----------|------------|-----------|-----------|
func (r *Renderer) buildHeader(name string, size uint64, ranges []Range) (string, error) {
	var sb strings.Builder

	// Helper to build a runes slice filled with the provided fill rune.
	makeRunes := func(n int, fill rune) []rune {
		s := make([]rune, n)
		for i := range s {
			s[i] = fill
		}
		return s
	}

	// Compose fileOffsetAxis with marker at 0%, 25%, 50%, 75%, 100% of size.
	fileOffsetAxisChars := makeRunes(r.plotWidth, '-')
	fileOffsetMarkers := []uint64{0, size / 4, size / 2, (size * 3) / 4, size}
	for _, off := range fileOffsetMarkers {
		if p, err := mapCoord(off, size, r.plotWidth); err == nil {
			fileOffsetAxisChars[p] = '|'
		}
	}

	// File offset labels, placed above the fileOffsetAxis at the size markers.
	fileOffsetLabels := makeRunes(r.plotWidth, ' ')
	for _, off := range fileOffsetMarkers {
		p, err := mapCoord(off, size, r.plotWidth)
		if err != nil {
			return "", err
		}

		offsetLabel := humanReadable(off)
		// Center fileOffsetLabel around the fileOffsetMarker.
		start := max(p-len(offsetLabel)/2, 0)
		if start+len(offsetLabel) > r.plotWidth {
			start = max(r.plotWidth-len(offsetLabel), 0)
		}
		copy(fileOffsetLabels[start:], []rune(offsetLabel))
	}

	// Filename line.
	sb.WriteString(fmt.Sprintf("Name: %s\n", name))

	// IO stats.
	sb.WriteString(r.buildStats(ranges))

	// Fileoffset labels just above the fileOffsetAxis.
	sb.WriteString(strings.Repeat(" ", r.labelWidth))
	if r.pad > 0 {
		sb.WriteString(strings.Repeat(" ", r.pad))
	}
	sb.WriteString(string(fileOffsetLabels))
	sb.WriteByte('\n')

	// labelHeader ("[offset,len)") and horizontal tick line.
	sb.WriteString(labelHeader)
	if r.labelWidth > len(labelHeader) {
		sb.WriteString(strings.Repeat(" ", r.labelWidth-len(labelHeader)))
	}
	if r.pad > 0 {
		sb.WriteString(strings.Repeat(" ", r.pad))
	}
	sb.WriteString(string(fileOffsetAxisChars))
	sb.WriteByte('\n')

	return sb.String(), nil
}

// buildRow composes a single plotted row (label + plot cells) for the given range.
func (r *Renderer) buildRow(size uint64, rg Range) (string, error) {
	var sb strings.Builder

	// Validate range: do not normalize or clamp; return an error on unexpected values.
	if rg.Start > rg.End {
		return "", fmt.Errorf("invalid range: start > end: [%d,%d)", rg.Start, rg.End)
	}
	if rg.End > size {
		return "", fmt.Errorf("range extends beyond file size: [%d,%d) size=%d", rg.Start, rg.End, size)
	}

	// Build plotting row
	cells := make([]string, r.plotWidth)
	for j := range cells {
		cells[j] = emptyChar
	}

	s := rg.Start
	e := rg.End - 1 // make end inclusive for plotting

	// Map start/end to columns. Reserve column 0 as a separator when possible by
	// mapping into [1, plotWidth-1] when plotWidth > 1.
	cs, err := mapCoord(s, size, r.plotWidth-1)
	if err != nil {
		return "", err
	}
	ce, err := mapCoord(e, size, r.plotWidth-1)
	if err != nil {
		return "", err
	}
	cs = cs + 1
	ce = ce + 1

	for c := cs; c <= ce; c++ {
		cells[c] = blockChar
	}

	// Place a vertical separator glyph in column 0.
	if r.plotWidth > 0 && cells[0] == emptyChar {
		cells[0] = "|"
	}

	// Compose label and write.
	label := fmt.Sprintf("[%d,%s)", s, humanReadable(e-s+1))
	if len(label) > r.labelWidth {
		label = label[:r.labelWidth]
	}
	if len(label) < r.labelWidth {
		label = label + strings.Repeat(" ", r.labelWidth-len(label))
	}
	sb.WriteString(label)
	if r.pad > 0 {
		sb.WriteString(strings.Repeat(" ", r.pad))
	}

	// Write chart.
	sb.WriteString(strings.Join(cells, ""))

	return sb.String(), nil
}

// mapCoord maps an offset in [0, size) to a column in [0, plotWidth).
// If size == 0 returns 0. plotWidth must be >0.
func mapCoord(offset, size uint64, plotWidth int) (int, error) {
	if plotWidth <= 0 || size == 0 {
		return 0, fmt.Errorf("invalid arguments to mapCoord")
	}
	frac := float64(offset) / float64(size)
	col := int(math.Floor(frac * float64(plotWidth)))
	if col < 0 {
		return 0, nil
	}
	if col >= plotWidth {
		return plotWidth - 1, nil
	}
	return col, nil
}
