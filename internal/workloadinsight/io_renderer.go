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
	PlotWidth  int // number of columns used for plotting area
	LabelWidth int // width of left label column (0 = auto)
	Pad        int // spaces between label and plot
}

func NewRenderer() (*Renderer, error) {
	return NewRendererWithSettings(80, len(labelHeader), 2)
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
		PlotWidth:  plotWidth,
		LabelWidth: labelWidth,
		Pad:        pad,
	}, nil
}

// Render renders the given ranges for a single file of the specified size
// and returns the ASCII representation as a string.
func (r *Renderer) Render(name string, size uint64, ranges []Range) (string, error) {
	var sb strings.Builder

	sb.WriteString(r.buildHeader(name, size))

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
	return sb.String(), nil
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
func (r *Renderer) buildHeader(name string, size uint64) string {
	var sb strings.Builder

	// Helper to build a runes slice filled with the provided fill rune.
	makeRunes := func(n int, fill rune) []rune {
		s := make([]rune, n)
		for i := range s {
			s[i] = fill
		}
		return s
	}

	// Compose size separator with marker at 0%, 25%, 50%, 75%, 100% of size.
	tickChars := makeRunes(r.PlotWidth, '-')
	ticks := []uint64{0, size / 4, size / 2, (size * 3) / 4, size}
	for _, off := range ticks {
		p := mapCoord(off, size, r.PlotWidth)
		if p >= 0 && p < r.PlotWidth {
			tickChars[p] = '|'
		}
	}

	// Numeric label line (place labels under ticks)
	labelLine := makeRunes(r.PlotWidth, ' ')
	for _, off := range ticks {
		p := mapCoord(off, size, r.PlotWidth)
		lab := humanReadable(off)
		// center label around the tick position
		start := p - len(lab)/2
		if start < 0 {
			start = 0
		}
		if start+len(lab) > r.PlotWidth {
			start = r.PlotWidth - len(lab)
			if start < 0 {
				start = 0
			}
		}
		copy(labelLine[start:], []rune(lab))
	}

	// Filename line
	sb.WriteString(fmt.Sprintf("Name: %s\n", name))

	// Numeric labels below filename
	sb.WriteString(strings.Repeat(" ", r.LabelWidth))
	if r.Pad > 0 {
		sb.WriteString(strings.Repeat(" ", r.Pad))
	}
	sb.WriteString(string(labelLine))
	sb.WriteByte('\n')

	// labelHeader ("[offset,len)") and horizontal tick line.
	sb.WriteString(labelHeader)
	if r.LabelWidth > len(labelHeader) {
		sb.WriteString(strings.Repeat(" ", r.LabelWidth-len(labelHeader)))
	}
	if r.Pad > 0 {
		sb.WriteString(strings.Repeat(" ", r.Pad))
	}
	sb.WriteString(string(tickChars))
	sb.WriteByte('\n')

	return sb.String()
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

	s := rg.Start
	e := rg.End

	// Build plotting row
	cells := make([]string, r.PlotWidth)
	for j := range cells {
		cells[j] = emptyChar
	}

	// Map start/end to columns. Reserve column 0 as a separator when possible by
	// mapping into [1, PlotWidth-1] when PlotWidth > 1.
	var cs, ce int
	if r.PlotWidth > 1 {
		cs = mapCoord(s, size, r.PlotWidth-1) + 1
		ce = mapCoord(e, size, r.PlotWidth-1) + 1
	} else {
		cs = mapCoord(s, size, r.PlotWidth)
		ce = mapCoord(e, size, r.PlotWidth)
	}

	// Ensure at least one visible column is set for very small ranges.
	if cs == ce {
		if cs < r.PlotWidth-1 {
			ce = cs + 1
		} else if cs > 0 {
			cs = cs - 1
		}
	}

	// Clamp column indices to valid range and fill cells.
	if cs < 0 {
		cs = 0
	}
	if ce >= r.PlotWidth {
		ce = r.PlotWidth - 1
	}
	for c := cs; c <= ce; c++ {
		cells[c] = blockChar
	}

	// Place a vertical separator glyph in column 0.
	if r.PlotWidth > 0 && cells[0] == emptyChar {
		cells[0] = "|"
	}

	// Compose label and write.
	label := fmt.Sprintf("[%d,%d)", s, e-s)
	if len(label) > r.LabelWidth {
		label = label[:r.LabelWidth]
	}
	if len(label) < r.LabelWidth {
		label = label + strings.Repeat(" ", r.LabelWidth-len(label))
	}
	sb.WriteString(label)
	if r.Pad > 0 {
		sb.WriteString(strings.Repeat(" ", r.Pad))
	}

	// Write chart.
	sb.WriteString(strings.Join(cells, ""))

	return sb.String(), nil
}

// mapCoord maps an offset in [0,size] to a column in [0,plotWidth-1].
// If size == 0 returns 0. plotWidth must be >0.
func mapCoord(offset, size uint64, plotWidth int) int {
	if plotWidth <= 0 || size == 0 {
		return 0
	}
	frac := float64(offset) / float64(size)
	col := int(frac*float64(plotWidth-1) + 0.5)
	if col < 0 {
		return 0
	}
	if col >= plotWidth {
		return plotWidth - 1
	}
	return col
}
