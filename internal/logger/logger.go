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
	"log/slog"
	"log/syslog"
	"math"
	"os"
	"path"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"

	"github.com/google/uuid"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
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

// Tracef prints the message with TRACE severity in the specified format.
func Tracef(format string, v ...any) {
	defaultLogger.Log(context.Background(), LevelTrace, fmt.Sprintf(format, v...))
}

// Debugf prints the message with DEBUG severity in the specified format.
func Debugf(format string, v ...any) {
	defaultLogger.Debug(fmt.Sprintf(format, v...))
}

// Infof prints the message with INFO severity in the specified format.
func Infof(format string, v ...any) {
	defaultLogger.Info(fmt.Sprintf(format, v...))
}

// Info prints the message with info severity.
func Info(message string, args ...any) {
	defaultLogger.Info(message, args...)
}

// Warnf prints the message with WARNING severity in the specified format.
func Warnf(format string, v ...any) {
	defaultLogger.Warn(fmt.Sprintf(format, v...))
}

// Errorf prints the message with ERROR severity in the specified format.
func Errorf(format string, v ...any) {
	defaultLogger.Error(fmt.Sprintf(format, v...))
}

// Error prints the message with ERROR severity.
func Error(error string) {
	defaultLogger.Error(error)
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
	var programLevel = new(slog.LevelVar)
	logger := slog.New(f.handler(programLevel, ""))
	slog.SetDefault(logger)
	setLoggingLevel(level, programLevel)
	return logger
}

func loggerAttr(fsName string) []slog.Attr {
	return []slog.Attr{slog.String(MountIDKey, MountInstanceID(fsName))}
}

// create a new logger with mountInstanceID set as custom attribute on logger.
func (f *loggerFactory) newLoggerWithMountInstanceID(level, fsName string) *slog.Logger {
	var programLevel = new(slog.LevelVar)
	logger := slog.New(f.handler(programLevel, "").WithAttrs(loggerAttr(fsName)))
	slog.SetDefault(logger)
	setLoggingLevel(level, programLevel)
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

// MetricType is a custom type to define our latency categories.
type MetricType string

const (
	READ_CALL_BLOCK_WAIT MetricType = "read_call_block_wait"
	READ_CALL_LAT        MetricType = "read_call_latency"
	BLOCK_DOWNLOAD_LAT   MetricType = "block_download_latency"
)

// metricData is an internal struct used only for passing data through the channel
type metricData struct {
	mType    MetricType
	duration int64
}

var (
	// 1. The Global Storage
	// We need a RWMutex because you might want to READ this map
	// while the background worker is writing to it.
	latencies = make(map[MetricType][]int64)
	mu        sync.RWMutex

	// 2. The Global Channel (The Buffer)
	// Buffer size 1000 means the first 1000 Add calls are instant/non-blocking
	latencyCh = make(chan metricData, 1000000)
)

// --- Core Logic ---

// init runs automatically when the program starts.
// We use it to start the background consumer immediately.
func init() {
	go processLatencies()
}

// processLatencies is the "Consumer".
// It runs in the background to drain the channel and update the map.
func processLatencies() {
	for data := range latencyCh {
		// We lock here, in the background, so the main program flow isn't slowed down.
		mu.Lock()
		latencies[data.mType] = append(latencies[data.mType], data.duration)
		mu.Unlock()
	}
}

// Add is the "Producer".
// It is THREAD SAFE and NON-BLOCKING (unless buffer is full).
func Add(metricType MetricType, dur time.Duration) {
	// Create the data packet
	data := metricData{
		mType:    metricType,
		duration: dur.Microseconds(),
	}

	// Send to the global channel.
	// This happens instantly; we do NOT wait for the lock here.
	latencyCh <- data
}

var readFileTotal int64
var blockTimeTotal int64

func Print(metricType MetricType) {
	// 1. Get the reference to the original data
	originalData, ok := latencies[metricType]
	if !ok || len(originalData) == 0 {
		Errorf("Metric %q has no data to print.", metricType)
		return
	}

	count := len(originalData)

	// 2. Calculate Sum/Min/Max using the original data (Order doesn't matter here)
	var sum int64 = 0
	var min int64 = originalData[0]
	var max int64 = originalData[0]

	for _, val := range originalData {
		sum += val
		if val < min {
			min = val
		}
		if val > max {
			max = val
		}
	}

	// 3. CRITICAL FIX: Create a copy for percentile calculation
	// We need a sorted list for P50/P99, but we must not touch originalData.
	sortedData := slices.Clone(originalData)
	slices.Sort(sortedData)

	p50Index := count / 2
	p99Index := int(float64(count) * 0.99)

	if p99Index >= count {
		p99Index = count - 1
	}

	avgMs := int64(float64(sum) / float64(count))
	minMs := min
	maxMs := max

	// Use the sorted copy for percentiles
	p50Ms := sortedData[p50Index]
	p99Ms := sortedData[p99Index]

	Infof("--- Latency Metrics for **%s** ---", metricType)
	Infof("  Count: %d", count)
	Infof("  Min:   %d us", minMs)
	Infof("  Avg:   %d us", avgMs)
	Infof("  P50:   %d us", p50Ms)
	Infof("  P99:   %d us", p99Ms)
	Infof("  Max:   %d us", maxMs)
	Infof("  Sum:   %d us", sum)
	Infof("======================================")
	if metricType == READ_CALL_BLOCK_WAIT {
		blockTimeTotal = sum
	}
	if metricType == READ_CALL_LAT {
		readFileTotal = sum
	}
}

// PrintAll iterates over all stored MetricTypes and calls Print for each one.
func PrintAll() {
	Info("=== Printing All Performance Metrics ===")
	if len(latencies) == 0 {
		Info("No metrics have been recorded yet.")
		return
	}

	// Iterate over the keys (MetricType) in the latencies map.
	// The iteration order of Go maps is not guaranteed to be the same
	// from one execution to the next.
	for metricType := range latencies {
		Print(metricType)
		dir, err := os.Getwd()
		if err != nil {
			Errorf("Could not get current directory for plotting: %v", err)
		} else {
			PlotLatencies(metricType, dir)
		}
	}
	Info("======================================")

	// Calculate the percentage: (Blocked Time / Total Read Time) * 100
	// We convert the integers to float64 for accurate division.
	percentageBlocked := (float64(blockTimeTotal) / float64(readFileTotal)) * 100

	// --- Output Results ---
	Infof("--- Performance Metrics Summary ---")
	Infof("Total Blocked/Wait Time (us): %d", blockTimeTotal)
	Infof("Total Read Call Time (us):    %d", readFileTotal)
	Infof("-----------------------------------")
	Infof("Percentage of time the reader was blocked: %.2f%%", percentageBlocked)
}

// yErrorPoints is a helper struct for plotter.YErrorBars to draw vertical lines
// from a minimum y-value up to the data point.
type yErrorPoints struct {
	plotter.XYs
	yMin float64
}

// YError implements the plotter.YErrorer interface.
func (p yErrorPoints) YError(i int) (float64, float64) {
	y := p.XYs[i].Y
	// We want the error bar to go from yMin up to y.
	// The bar is drawn from (y - low) to (y + high).
	// So, y - low = yMin  => low = y - yMin
	// And y + high = y   => high = 0
	low := y - p.yMin
	if low < 0 {
		low = 0 // Should not happen if all y >= yMin
	}
	return low, 0
}

// customLogTicker is a plot.Ticker that generates tick marks for a log scale.
// It provides more labels than the default LogTicker by labeling ticks at
// multiples of 1, 2, and 5 for each power of 10.
type customLogTicker struct{}

// Ticks returns Ticks in a log scale.
func (customLogTicker) Ticks(min, max float64) []plot.Tick {
	if max <= min {
		return nil
	}

	// format returns a string representation of the tick value.
	format := func(v float64) string {
		return fmt.Sprintf("%.0f", v)
	}

	var ticks []plot.Tick
	val := math.Pow(10, math.Floor(math.Log10(min)))

	for val <= max {
		for i := 1; i < 10; i++ {
			tickVal := val * float64(i)
			if tickVal > max {
				break
			}
			if tickVal >= min {
				tick := plot.Tick{Value: tickVal}
				if i == 1 || i == 2 || i == 5 { // Label 1x, 2x, 5x ticks
					tick.Label = format(tickVal)
				}
				ticks = append(ticks, tick)
			}
		}
		if val > max/10 { // Avoid overflow and infinite loops
			break
		}
		val *= 10
	}
	return ticks
}

func PlotLatencies(metricType MetricType, dir string) {
	data, ok := latencies[metricType]
	if !ok || len(data) == 0 {
		Errorf("Metric %q has no data to plot.\n", metricType)
		return
	}

	// Define defaults if they aren't global
	const plotFloorValue = 1.0
	floorCount := 0

	// 1. Prepare Plot Points
	pts := make(plotter.XYs, len(data))
	for i, latencyUs := range data {
		var plotValue float64

		// CRITICAL: Replace non-positive values (<= 0) with 1 µs for the log scale.
		if latencyUs <= 0 {
			plotValue = plotFloorValue
			floorCount++
		} else {
			plotValue = float64(latencyUs)
		}

		// X-axis: Index of the measurement
		pts[i].X = float64(i + 1)

		// --- FIX IS HERE ---
		// Previously: pts[i].Y = float64(latencyUs) // This caused the panic on 0 values
		// Now: Use the sanitized 'plotValue' variable
		pts[i].Y = plotValue
	}

	// 2. Create the Plot
	p := plot.New()

	p.Title.Text = fmt.Sprintf("Latency Over Time: %s", metricType)
	p.X.Label.Text = "Measurement Index"

	// Y-axis Label updated to Microseconds
	p.Y.Label.Text = "Latency (µs) - Log Scale"

	// 3. Configure the Y-axis to be logarithmic
	p.Y.Scale = plot.LogScale{}

	// Use the custom ticker to get more labels on the Y-axis.
	p.Y.Tick.Marker = customLogTicker{}

	// Optional: set the Min to the floor value so the axis starts cleanly
	p.Y.Min = plotFloorValue

	// 4. Create vertical green lines for each data point.
	errBars, err := plotter.NewYErrorBars(yErrorPoints{pts, plotFloorValue})
	if err != nil {
		Errorf("Could not create YErrorBars plot: %v\n", err)
		return
	}
	errBars.LineStyle.Color = color.RGBA{G: 255, A: 255} // Green
	errBars.CapWidth = 0                                 // No caps on the error bars

	p.Add(errBars)

	// 5. Save the plot to a file
	filename := path.Join(dir, fmt.Sprintf("%s.png", metricType))
	if err := p.Save(10*vg.Inch, 6*vg.Inch, filename); err != nil {
		Errorf("Could not save plot to file %q: %v\n", filename, err)
	} else {
		Infof("Successfully created plot for %q at %q\n", metricType, filename)
		if floorCount > 0 {
			Infof("Note: %d data points were <= 0 and clamped to %0.1f µs for log scaling.\n", floorCount, plotFloorValue)
		}
	}
}
