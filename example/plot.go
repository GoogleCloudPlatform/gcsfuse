package main

import (
        "fmt"
        "image/color"
        "log"
        "math"

        "gonum.org/v1/plot"
        "gonum.org/v1/plot/plotter"
        "gonum.org/v1/plot/vg"
)

// The main drawback of using the gonum/plot library is that it does not support
// adding new graphs to the same file. so, we cannot append graphs to the same
// file. Each time we run the program, it will overwrite the previous graph.
// A more advanced plotting library or custom implementation would be needed
// to support appending graphs to the same file.

// ReadRange represents a single read operation [Start, End) offset in the file.
type ReadRange struct {
        Start, End int64
}

func main() {
        // Example Inputs
        fileSize := int64(20000) // Total size of the file
        readRanges := []ReadRange{
                {Start: 0, End: 1000},     // Should be at the top (Y ~ numRanges)
                {Start: 2000, End: 2500},
                {Start: 500, End: 1500},
                {Start: 18000, End: 19500},
                {Start: 2100, End: 2200},
                {Start: 8000, End: 12000}, // Should be at the bottom (Y ~ 1)
        }

        numRanges := len(readRanges)
        if numRanges == 0 {
                fmt.Println("No read ranges to plot.")
                return
        }

        p := plot.New()

        p.Title.Text = "File Read Pattern (Top-Down)"
        p.X.Label.Text = "File Offset"
        p.Y.Label.Text = "Read Operation Index"

        // X-Axis Configuration
        p.X.Min = 0
        p.X.Max = float64(fileSize)
        p.X.Tick.Marker = plot.DefaultTicks{}

        // Y-Axis Configuration
        p.Y.Min = 0.5
        p.Y.Max = float64(numRanges) + 0.5

        // Custom Y ticks for each read operation (1 at the top)
        var yTicks []plot.Tick
        for i := 1; i <= numRanges; i++ {
                // Operation i (1-indexed) maps to Y value (numRanges - i + 1)
                yValue := float64(numRanges - i + 1)
                yTicks = append(yTicks, plot.Tick{Value: yValue, Label: fmt.Sprintf("%d", i)})
        }
        p.Y.Tick.Marker = plot.ConstantTicks(yTicks)
        // p.Y.Grid.Liner = nil

        const barHeight = 0.8

        for i, rr := range readRanges {
                if rr.Start > rr.End {
                        log.Printf("Warning: Read range %d has Start > End: [%d, %d)", i+1, rr.Start, rr.End)
                        continue
                }
                if rr.End > fileSize {
                        log.Printf("Warning: Read range %d extends beyond file size: [%d, %d)", i+1, rr.Start, rr.End)
                        rr.End = fileSize
                }

                // Y-center for top-down: first item (i=0) has highest Y value
                yCenter := float64(numRanges - i)
                yMin := yCenter - barHeight/2
                yMax := yCenter + barHeight/2

                pts := plotter.XYs{
                        {X: float64(rr.Start), Y: yMin},
                        {X: float64(rr.End), Y: yMin},
                        {X: float64(rr.End), Y: yMax},
                        {X: float64(rr.Start), Y: yMax},
                }

                poly, err := plotter.NewPolygon(pts)
                if err != nil {
                        log.Panicf("Could not create polygon for range %d: %v", i+1, err)
                }
                poly.Color = color.RGBA{R: 30, G: 144, B: 255, A: 255} // DodgerBlue
                poly.LineStyle.Width = 0

                p.Add(poly)
        }

        // --- Dimensions ---
        width := 12 * vg.Inch
        height := vg.Length(math.Max(2.0, float64(numRanges)*0.4)) * vg.Inch
        maxHeight := 20 * vg.Inch
        if height > maxHeight {
                height = maxHeight
                fmt.Printf("Warning: Too many read ranges (%d). Graph height capped, rows will be compressed.\n", numRanges)
        }

        if err := p.Save(width, height, "file_read_pattern_topdown.png"); err != nil {
                log.Panicf("Could not save plot: %v", err)
        }
        fmt.Println("Plot saved to file_read_pattern_topdown.png")
}
