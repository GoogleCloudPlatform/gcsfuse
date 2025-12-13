// Copyright 2020 Google LLC
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

package logger

import (
	"context"
	"fmt"
	"image/color"
	"io"
	"log"
	"log/slog"
	"log/syslog"
	"os"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Syslog file contains logs from all different programs running on the VM.
// ProgramName is prefixed to all the logs written to syslog. This constant is
// used to filter the logs from syslog and write it to respective log files -
// gcsfuse.log in case of GCSFuse.
const (
	ProgramName             = "gcsfuse"
	GCSFuseInBackgroundMode = "GCSFUSE_IN_BACKGROUND_MODE"
	MountUUIDEnvKey         = "GCSFUSE_MOUNT_UUID"
	MountIDKey              = "mount-id" // Combination of fsName and GCSFUSE_MOUNT_UUID
	textFormat              = "text"
	// Max possible length can be 32 as UUID has 32 characters excluding 4 hyphens.
	mountUUIDLength = 8
)

var (
	defaultLoggerFactory *loggerFactory
	defaultLogger        *slog.Logger
	mountUUID            string
	setupMountUUIDOnce   sync.Once
	programLevel         = new(slog.LevelVar)
)

// InitLogFile initializes the logger factory to create loggers that print to
// a log file, with MountInstanceID set as a custom attribute.
// In case of empty file, it starts writing the log to syslog file, which
// is eventually filtered and redirected to a fixed location using syslog
// config.
// Here, background true means, this InitLogFile has been called for the
// background daemon.
func InitLogFile(newLogConfig cfg.LoggingConfig, fsName string) error {
	var f *os.File
	var sysWriter *syslog.Writer
	var fileWriter *lumberjack.Logger
	var err error
	if newLogConfig.FilePath != "" {
		f, err = os.OpenFile(
			string(newLogConfig.FilePath),
			os.O_WRONLY|os.O_CREATE|os.O_APPEND,
			0644,
		)
		if err != nil {
			return err
		}
		fileWriter = &lumberjack.Logger{
			Filename:   f.Name(),
			MaxSize:    int(newLogConfig.LogRotate.MaxFileSizeMb),
			MaxBackups: int(newLogConfig.LogRotate.BackupFileCount),
			Compress:   newLogConfig.LogRotate.Compress,
		}
	} else {
		if _, ok := os.LookupEnv(GCSFuseInBackgroundMode); ok {
			// Priority consist of facility and severity, here facility to specify the
			// type of system that is logging the message to syslog and severity is log-level.
			// User applications are allowed to take facility value between LOG_LOCAL0
			// to LOG_LOCAL7. We are using LOG_LOCAL7 as facility and LOG_DEBUG to write
			// debug messages.

			// Suppressing the error while creating the syslog, although logger will
			// be initialised with stdout/err, log will be printed anywhere. Because,
			// in this case gcsfuse will be running as daemon.
			sysWriter, _ = syslog.New(syslog.LOG_LOCAL7|syslog.LOG_DEBUG, ProgramName)
		}
	}

	defaultLoggerFactory = &loggerFactory{
		file:       f,
		sysWriter:  sysWriter,
		fileWriter: fileWriter,
		format:     newLogConfig.Format,
		level:      string(newLogConfig.Severity),
		logRotate:  newLogConfig.LogRotate,
	}
	defaultLogger = defaultLoggerFactory.newLoggerWithMountInstanceID(string(newLogConfig.Severity), fsName)

	return nil
}

// init initializes the logger factory to use stdout and stderr.
func init() {
	logConfig := cfg.DefaultLoggingConfig()
	defaultLoggerFactory = &loggerFactory{
		file:      nil,
		format:    logConfig.Format,
		level:     string(logConfig.Severity), // setting log level to INFO by default
		logRotate: logConfig.LogRotate,
	}
	defaultLogger = defaultLoggerFactory.newLogger(cfg.INFO)
}

// generateMountUUID generates a random string of size from UUID.
func generateMountUUID(size int) (string, error) {
	if size <= 0 {
		return "", fmt.Errorf("requested size for MountUUID must be positive, but got %d", size)
	}
	uuid := uuid.New()
	uuidStr := strings.ReplaceAll(uuid.String(), "-", "")
	if size > len(uuidStr) {
		return "", fmt.Errorf("UUID is smaller than requested size %d for MountUUID, UUID: %s", size, uuidStr)
	}
	return uuidStr[:size], nil
}

// setupMountUUID handles the retrieval of mountUUID if GCSFuse is in
// background mode or generates one if running in foreground mode.
func setupMountUUID() {
	if _, ok := os.LookupEnv(GCSFuseInBackgroundMode); ok {
		// If GCSFuse is in background mode then look for the GCSFUSE_MOUNT_UUID in env which was set by the caller of demonize run.
		if mountUUID, ok = os.LookupEnv(MountUUIDEnvKey); !ok || mountUUID == "" {
			Fatal("Could not retrieve %s env variable or it's empty.", MountUUIDEnvKey)
		}
		return
	}
	// If GCSFuse is not running in the background mode then generate a random UUID.
	var err error
	if mountUUID, err = generateMountUUID(mountUUIDLength); err != nil {
		Fatal("Could not generate MountUUID of length %d, err: %v", mountUUIDLength, err)
	}
}

// MountUUID returns a unique ID for the current GCSFuse mount,
// ensuring the ID is initialized only once. On the first call, it either
// generates a random ID (foreground mode) or retrieves it from the
// GCSFUSE_MOUNT_UUID environment variable (background mode).
// Subsequent calls return the same cached ID.
func MountUUID() string {
	setupMountUUIDOnce.Do(setupMountUUID)
	return mountUUID
}

// MountInstanceID returns the InstanceID of current gcsfuse mount.
// This is combination of `fsName` + MountUUID.
// Note: fsName is passed here explicitly, as logger package doesn't know about fsName
// when MountInstanceID method is invoked.
func MountInstanceID(fsName string) string {
	return fmt.Sprintf("%s-%s", fsName, MountUUID())
}

// UpdateDefaultLogger updates the log format and creates a new logger with MountInstanceID set as custom attribute.
func UpdateDefaultLogger(format, fsName string) {
	defaultLoggerFactory.format = format
	defaultLogger = defaultLoggerFactory.newLoggerWithMountInstanceID(defaultLoggerFactory.level, fsName)
}
func TraceEnabled() bool {
	return LevelTrace >= programLevel.Level()
}

// Tracef prints the message with TRACE severity in the specified format.
func Tracef(format string, v ...any) {
	if LevelTrace >= programLevel.Level() {
		defaultLogger.Log(context.Background(), LevelTrace, fmt.Sprintf(format, v...))
	}
}

// Debugf prints the message with DEBUG severity in the specified format.
func Debugf(format string, v ...any) {
	if LevelDebug >= programLevel.Level() {
		defaultLogger.Debug(fmt.Sprintf(format, v...))
	}
}

// Infof prints the message with INFO severity in the specified format.
func Infof(format string, v ...any) {
	if LevelInfo >= programLevel.Level() {
		defaultLogger.Info(fmt.Sprintf(format, v...))
	}
}

// Info prints the message with info severity.
func Info(message string, args ...any) {
	if LevelInfo >= programLevel.Level() {
		defaultLogger.Info(message, args...)
	}
}

// Warnf prints the message with WARNING severity in the specified format.
func Warnf(format string, v ...any) {
	if LevelWarn >= programLevel.Level() {
		defaultLogger.Warn(fmt.Sprintf(format, v...))
	}
}

// Errorf prints the message with ERROR severity in the specified format.
func Errorf(format string, v ...any) {
	if LevelError >= programLevel.Level() {
		defaultLogger.Error(fmt.Sprintf(format, v...))
	}
}

// Error prints the message with ERROR severity.
func Error(error string) {
	if LevelError >= programLevel.Level() {
		defaultLogger.Error(error)
	}
}

// Fatal prints an error log and exits with non-zero exit code.
func Fatal(format string, v ...any) {
	Errorf(format, v...)
	Error(string(debug.Stack()))
	os.Exit(1)
}

type loggerFactory struct {
	// If nil, log to stdout or stderr. Otherwise, log to this file.
	file       *os.File
	sysWriter  *syslog.Writer
	format     string
	level      string
	logRotate  cfg.LogRotateLoggingConfig
	fileWriter *lumberjack.Logger
}

func (f *loggerFactory) newLogger(level string) *slog.Logger {
	// create a new logger
	logger := slog.New(f.handler(programLevel, ""))
	slog.SetDefault(logger)
	setLoggingLevel(level)
	return logger
}

func loggerAttr(fsName string) []slog.Attr {
	return []slog.Attr{slog.String(MountIDKey, MountInstanceID(fsName))}
}

// create a new logger with mountInstanceID set as custom attribute on logger.
func (f *loggerFactory) newLoggerWithMountInstanceID(level, fsName string) *slog.Logger {
	logger := slog.New(f.handler(programLevel, "").WithAttrs(loggerAttr(fsName)))
	slog.SetDefault(logger)
	setLoggingLevel(level)
	return logger
}

func (f *loggerFactory) createJsonOrTextHandler(writer io.Writer, levelVar *slog.LevelVar, prefix string) slog.Handler {
	if f.format == textFormat {
		return slog.NewTextHandler(writer, getHandlerOptions(levelVar, prefix, f.format))
	}
	return slog.NewJSONHandler(writer, getHandlerOptions(levelVar, prefix, f.format))
}

func (f *loggerFactory) handler(levelVar *slog.LevelVar, prefix string) slog.Handler {
	if f.fileWriter != nil {
		return f.createJsonOrTextHandler(f.fileWriter, levelVar, prefix)
	}

	if f.sysWriter != nil {
		return f.createJsonOrTextHandler(f.sysWriter, levelVar, prefix)
	}
	return f.createJsonOrTextHandler(os.Stdout, levelVar, prefix)
}

// StackGrowthData holds the limit and usage data for stack growth analysis.
type StackGrowthData struct {
	Label string
	Limit []int
	Usage []int
}

// Add appends the usage and limit to the StackGrowthData.
//
//go:nosplit
func (s *StackGrowthData) Add(usage, limit int) {
	s.Usage = append(s.Usage, usage)
	s.Limit = append(s.Limit, limit)
}

var (
	// Before holds the stack growth data before the operation.
	Before = &StackGrowthData{Label: "before"}
	// After holds the stack growth data after the operation.
	After = &StackGrowthData{Label: "after"}
)

// ResetStackGrowthData resets the Before and After stack growth data.
func ResetStackGrowthData() {
	Before.Limit = nil
	Before.Usage = nil
	After.Limit = nil
	After.Usage = nil
}

// --- The Plotting Logic ---

// PlotStackGrowth generates a 2x2 grid of bar charts based on the global
// Before and After variables and saves it to the specified filename.
func PlotStackGrowth(filename string) {
	// Print statistics about the data points.
	Infof("--- Stack Growth Statistics ---")
	Infof("Before.Limit has %d data points.", len(Before.Limit))
	Infof("Before.Usage has %d data points.", len(Before.Usage))
	Infof("After.Limit has %d data points.", len(After.Limit))
	Infof("After.Usage has %d data points.", len(After.Usage))
	Infof("-----------------------------")
	// 1. Calculate Global Max to ensure a single scale across all 4 graphs
	globalMax := 0.0
	updateMax := func(vals []int) {
		for _, v := range vals {
			if float64(v) > globalMax {
				globalMax = float64(v)
			}
		}
	}

	updateMax(Before.Limit)
	updateMax(Before.Usage)
	updateMax(After.Limit)
	updateMax(After.Usage)

	// Avoid 0 max if data is empty, and add 10% padding for aesthetics
	if globalMax == 0 {
		globalMax = 10
	} else {
		globalMax = globalMax * 1.1
	}

	// 2. Helper to create a single plot with fixed scale
	createSubPlot := func(title string, data []int, c color.Color) *plot.Plot {
		p := plot.New()
		p.Title.Text = title
		p.Title.Padding = vg.Points(5)

		// Force the single scale
		p.Y.Min = 0
		p.Y.Max = globalMax

		p.X.Label.Text = "Index"
		p.Y.Label.Text = "Bytes"

		// Add ticks every 200 units on the Y-axis.
		var ticks []plot.Tick
		for i := 0.0; i <= globalMax; i += 200 {
			ticks = append(ticks, plot.Tick{
				Value: i,
				Label: fmt.Sprintf("%.0f", i),
			})
		}
		p.Y.Tick.Marker = plot.ConstantTicks(ticks)
		// Create Bar Chart
		// width is set to 2 points for visibility, adjust if data is huge
		bars, err := plotter.NewBarChart(intToValues(data), vg.Points(2))
		if err != nil {
			log.Printf("Error creating bar chart for %s: %v", title, err)
			return p
		}
		bars.LineStyle.Width = vg.Length(0) // No outline
		bars.Color = c

		p.Add(bars)
		return p
	}

	// 3. Generate the 4 plots
	// Colors: Blue, Orange, Green, Red
	p1 := createSubPlot("Limit (Before)", Before.Limit, color.RGBA{R: 0, G: 0, B: 255, A: 255})
	p2 := createSubPlot("Usage (Before)", Before.Usage, color.RGBA{R: 255, G: 165, B: 0, A: 255})
	p3 := createSubPlot("Limit (After)", After.Limit, color.RGBA{R: 0, G: 128, B: 0, A: 255})
	p4 := createSubPlot("Usage (After)", After.Usage, color.RGBA{R: 255, G: 0, B: 0, A: 255})

	// 4. Create the Image Canvas (2x2 Grid)
	const rows, cols = 2, 2
	// Canvas size: 1000x800 points
	c := vgimg.New(vg.Points(1000), vg.Points(800))
	dc := draw.New(c)

	// Create a tiled layout
	t := draw.Tiles{
		Rows:      rows,
		Cols:      cols,
		PadX:      vg.Points(20),
		PadY:      vg.Points(20),
		PadTop:    vg.Points(20),
		PadBottom: vg.Points(20),
		PadLeft:   vg.Points(20),
		PadRight:  vg.Points(20),
	}

	plots := [][]*plot.Plot{
		{p1, p2},
		{p3, p4},
	}

	// Align the plots on the canvas
	canvases := plot.Align(plots, t, dc)

	// Draw each plot onto its respective sub-canvas
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			plots[i][j].Draw(canvases[i][j])
		}
	}

	// 5. Save to File
	w, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to create file %s: %v", filename, err)
	}
	defer w.Close()

	png := vgimg.PngCanvas{Canvas: c}
	if _, err := png.WriteTo(w); err != nil {
		log.Fatalf("Failed to write to file %s: %v", filename, err)
	}

	fmt.Printf("Graph saved to %s\n", filename)
	ResetStackGrowthData()
}

// intToValues converts []int to plotter.Values (which is []float64)
func intToValues(data []int) plotter.Values {
	vs := make(plotter.Values, len(data))
	for i, v := range data {
		vs[i] = float64(v)
	}
	return vs
}
