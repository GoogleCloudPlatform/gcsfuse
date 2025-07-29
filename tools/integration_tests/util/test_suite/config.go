package test_suite

// TestConfig defines the structure for a single test run's configuration.
type TestConfig struct {
	MountedDirectory string   `yaml:"mounted_directory"`
	TestBucket       string   `yaml:"test_bucket"`
	LogFile          string   `yaml:"log_file,omitempty"`
	Flags            []string `yaml:"flags"`
	BucketType       []string `yaml:"bucket_type"`
}
