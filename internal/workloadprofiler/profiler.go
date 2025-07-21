package workloadprofiler

import (
	"context"
	"encoding/json"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

type ProfilerSource interface {
	// GetProfileData returns the current profile data.
	// This method should also reset the internal state so in the next
	// call it returns fresh data.
	GetProfileData() map[string]interface{}
}

var gWorkloadProfiler *WorkloadProfiler

// Please initialize the workload profiler in the init function of your application.
func init() {
	gWorkloadProfiler = newWorkloadProfiler(45*time.Second, yamlDumpingCallback) // Default to 60 seconds interval
	gWorkloadProfiler.Start()
	logger.Info("Workload Profiler initialized. Use NewWorkloadProfiler to create an instance.")
	AddProfilerSource(NewVMResourceSource())
	if gWorkloadProfiler != nil {
		runtime.SetFinalizer(gWorkloadProfiler, func(wp *WorkloadProfiler) {
			gWorkloadProfiler.Stop()
		})
	} else {
		logger.Error("Failed to initialize Workload Profiler. It will not be started.")
	}
}

// DumpCallback is a function type that handles the dumped profile data.
// It receives a copy of the profile data.
type DumpCallback func(data map[string]interface{}, dumpDir string)

// WorkloadProfiler collects and periodically dumps workload profile data.
// It is safe for concurrent use.
type WorkloadProfiler struct {
	profileData map[string]interface{}
	// dumpInterval is the time interval between profile dumps.
	dumpInterval time.Duration
	// dumpCallback is the function called to dump the profile data.
	dumpCallback DumpCallback

	profilerSources []ProfilerSource // List of sources to collect profile data from.

	dumpDir string // Directory where dumps are saved, if applicable.

	// mu protects access to profileData.
	mu sync.RWMutex
	// wg is used to wait for goroutines to finish on stop.
	wg sync.WaitGroup
	// ctx and cancel are used to manage the lifecycle of the running goroutines.
	ctx    context.Context
	cancel context.CancelFunc
}

// defaultDump is the default callback which prints the profile data as JSON
// to standard output.
func defaultDump(data map[string]interface{}, dumpDir string) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		logger.Errorf("Error marshalling profile data to JSON: %v", err)
		return
	}
	logger.Infof(string(jsonData))
}

// NewWorkloadProfiler creates a new WorkloadProfiler.
//
// dumpInterval specifies how often the profile data should be dumped.
// dumpCallback is the function that will be executed with the profile data. If nil,
// it defaults to printing the data to the console.
func newWorkloadProfiler(dumpInterval time.Duration, dumpCallback DumpCallback) *WorkloadProfiler {
	if dumpCallback == nil {
		dumpCallback = defaultDump
	}

	// Create a cancellable context to manage the lifecycle of the profiler's goroutines.
	ctx, cancel := context.WithCancel(context.Background())
	wp := &WorkloadProfiler{
		profileData:  make(map[string]interface{}),
		dumpInterval: dumpInterval,
		dumpCallback: dumpCallback,
		ctx:          ctx,
		cancel:       cancel,
	}

	var err error
	wp.dumpDir, err = os.UserHomeDir()
	if err != nil {
		logger.Errorf("Error getting home directory: %v", err)
		return nil
	}

	return wp
}

func AddProfilerSource(source ProfilerSource) {
	if gWorkloadProfiler == nil {
		logger.Error("Workload Profiler is not initialized. Cannot add profiler source.")
		return
	}
	gWorkloadProfiler.addProfilerSource(source)
}

func (p *WorkloadProfiler) addProfilerSource(source ProfilerSource) {
	p.mu.Lock()
	defer p.mu.Unlock()
	logger.Infof("Adding profiler source: %T", source)
	p.profilerSources = append(p.profilerSources, source)
}

// Start begins the periodic dumping of profile data in a separate goroutine.
// It is safe to call Start multiple times, but it will only start the profiler once.
func (p *WorkloadProfiler) Start() {
	p.wg.Add(1)
	go p.dumpLoop()
	logger.Info("Workload Profiler started.")
}

// Stop gracefully stops the profiler and waits for the dump goroutine to finish.
func (p *WorkloadProfiler) Stop() {
	// Signal the goroutines to stop.
	p.cancel()
	// Wait for all goroutines to acknowledge the stop signal and exit.
	p.wg.Wait()
	logger.Info("Workload Profiler stopped.")
}

// dumpLoop is the main loop for periodically dumping the profile data.
// This method is intended to be run as a goroutine.
func (p *WorkloadProfiler) dumpLoop() {
	defer p.wg.Done()
	ticker := time.NewTicker(p.dumpInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.mu.RLock()
			for _, source := range p.profilerSources {
				// Collect profile data from each source and merge it into the main profile data.
				data := source.GetProfileData()
				for k, v := range data {
					p.profileData[k] = v
				}
			}
			p.mu.RUnlock()
			copyProfiledData := p.GetProfileData()
			p.ResetProfileData()
			p.dumpCallback(copyProfiledData, p.dumpDir)
		case <-p.ctx.Done():
			// Context was cancelled, so we exit the loop.
			return
		}
	}
}

// GetProfileData returns a copy of the current profile data for inspection.
func (p *WorkloadProfiler) GetProfileData() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	dataCopy := make(map[string]interface{}, len(p.profileData))
	for k, v := range p.profileData {
		dataCopy[k] = v
	}
	return dataCopy
}

func (p *WorkloadProfiler) ResetProfileData() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.profileData = make(map[string]interface{})
}
