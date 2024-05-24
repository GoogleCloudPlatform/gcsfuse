// Copyright 2024 Google Inc. All Rights Reserved.
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

package cfg

import (
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestParsingSuccess(t *testing.T) {
	type TestConfig struct {
		OctalParam       Octal
		URLParam         url.URL
		BoolParam        bool
		StringParam      string
		IntParam         int
		FloatParam       float64
		DurationParam    time.Duration
		StringSliceParam []string
		IntSliceParam    []int
		LogSeverityParam LogSeverity
		ProtocolParam    Protocol
		PathParam        ResolvedPath
	}
	declareFlags := func() *flag.FlagSet {
		fs := flag.NewFlagSet("test", flag.ExitOnError)
		fs.String("urlParam", "", "")
		fs.String("octalParam", "0", "")
		fs.String("stringParam", "", "")
		fs.Int("intParam", 0, "")
		fs.Float64("floatParam", 0.0, "")
		fs.Duration("durationParam", 0*time.Nanosecond, "")
		fs.StringSlice("stringSliceParam", []string{}, "")
		fs.IntSlice("intSliceParam", []int{}, "")
		fs.Bool("boolParam", false, "")
		fs.String("logSeverityParam", "INFO", "")
		fs.String("protocolParam", "http1", "")
		fs.String("pathParam", "", "")
		return fs
	}

	bindFlags := func(fs *flag.FlagSet) *viper.Viper {
		v := viper.New()
		v.BindPFlag("URLParam", fs.Lookup("urlParam"))
		v.BindPFlag("OctalParam", fs.Lookup("octalParam"))
		v.BindPFlag("StringParam", fs.Lookup("stringParam"))
		v.BindPFlag("IntParam", fs.Lookup("intParam"))
		v.BindPFlag("FloatParam", fs.Lookup("floatParam"))
		v.BindPFlag("DurationParam", fs.Lookup("durationParam"))
		v.BindPFlag("StringSliceParam", fs.Lookup("stringSliceParam"))
		v.BindPFlag("IntSliceParam", fs.Lookup("intSliceParam"))
		v.BindPFlag("BoolParam", fs.Lookup("boolParam"))
		v.BindPFlag("LogSeverityParam", fs.Lookup("logSeverityParam"))
		v.BindPFlag("ProtocolParam", fs.Lookup("protocolParam"))
		v.BindPFlag("PathParam", fs.Lookup("pathParam"))
		return v
	}
	tests := []struct {
		name    string
		args    []string
		param   string
		setupFn func()
		testFn  func(*testing.T, TestConfig)
	}{
		{
			name: "Octal",
			args: []string{"--octalParam=755"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.Equal(t, Octal(0755), c.OctalParam)
			},
		},
		{
			name: "URL",
			args: []string{"--urlParam=http://abc.xyz"},
			testFn: func(t *testing.T, c TestConfig) {
				u, err := url.Parse("http://abc.xyz")
				if err != nil {
					t.Fatalf("Error while parsing URL: %v", err)
				}
				assert.Equal(t, *u, c.URLParam)
			},
		},
		{
			name: "Bool1",
			args: []string{"--boolParam"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.True(t, c.BoolParam)
			},
		},
		{
			name: "Bool2",
			args: []string{"--boolParam=true"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.True(t, c.BoolParam)
			},
		},
		{
			name: "Bool3",
			args: []string{"--boolParam=false"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.False(t, c.BoolParam)
			},
		},
		{
			name: "String",
			args: []string{"--stringParam=abc"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.Equal(t, "abc", c.StringParam)
			},
		},
		{
			name: "Int",
			args: []string{"--intParam=23"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.Equal(t, 23, c.IntParam)
			},
		},
		{
			name: "Float",
			args: []string{"--floatParam=2.5"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.Equal(t, 2.5, c.FloatParam)
			},
		},
		{
			name: "Duration",
			args: []string{"--durationParam=30s"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.Equal(t, 30*time.Second, c.DurationParam)
			},
		},
		{
			name: "StringSlice1",
			args: []string{"--stringSliceParam=a,b"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.ElementsMatch(t, []string{"a", "b"}, c.StringSliceParam)
			},
		},
		{
			name: "StringSlice2",
			args: []string{"--stringSliceParam=a", "--stringSliceParam=b"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.ElementsMatch(t, []string{"a", "b"}, c.StringSliceParam)
			},
		},
		{
			name: "IntSlice1",
			args: []string{"--intSliceParam=2,-11"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.ElementsMatch(t, []int{2, -11}, c.IntSliceParam)
			},
		},
		{
			name: "IntSlice2",
			args: []string{"--intSliceParam=3", "--intSliceParam=5"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.ElementsMatch(t, []int{3, 5}, c.IntSliceParam)
			},
		},
		{
			name: "LogSeverity",
			args: []string{"--logSeverityParam=WARNING"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.Equal(t, LogSeverity("WARNING"), c.LogSeverityParam)
			},
		},
		{
			name: "ResolvedPath1",
			args: []string{"--pathParam=~/test.txt"},
			testFn: func(t *testing.T, c TestConfig) {
				h, err := os.UserHomeDir()
				if assert.Nil(t, err) {
					assert.Equal(t, path.Join(h, "test.txt"), string(c.PathParam))
				}
			},
		},
		{
			name: "ResolvedPath2",
			setupFn: func() {
				os.Setenv("gcsfuse-parent-process-dir", "/a")
				t.Cleanup(func() { os.Unsetenv("gcsfuse-parent-process-dir") })
			},
			args: []string{"--pathParam=./test.txt"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.Equal(t, "/a/test.txt", string(c.PathParam))
			},
		},
	}

	for _, k := range tests {
		t.Run(k.name, func(t *testing.T) {
			if k.setupFn != nil {
				k.setupFn()
			}
			c := TestConfig{}
			fs := declareFlags()
			v := bindFlags(fs)
			args := []string{"test"}
			args = append(args, k.args...)
			err := fs.Parse(args)
			if err != nil {
				t.Fatalf("Flag parsing failed: %v", err)
			}

			err = v.Unmarshal(&c, viper.DecodeHook(DecodeHook()))

			if assert.Nil(t, err) {
				k.testFn(t, c)
			}
		})
	}
}

func TestParsingError(t *testing.T) {
	type TestConfig struct {
		OctalParam       Octal
		URLParam         url.URL
		LogSeverityParam LogSeverity
		ProtocolParam    Protocol
	}
	declareFlags := func() *flag.FlagSet {
		fs := flag.NewFlagSet("test", flag.ExitOnError)
		fs.String("octalParam", "0", "")
		fs.String("urlParam", "", "")
		fs.String("logSeverityParam", "INFO", "")
		fs.String("protocolParam", "http1", "")
		return fs
	}
	bindFlags := func(fs *flag.FlagSet) *viper.Viper {
		v := viper.New()
		v.BindPFlag("OctalParam", fs.Lookup("octalParam"))
		v.BindPFlag("URLParam", fs.Lookup("urlParam"))
		v.BindPFlag("LogSeverityParam", fs.Lookup("logSeverityParam"))
		v.BindPFlag("ProtocolParam", fs.Lookup("protocolParam"))
		return v
	}
	tests := []struct {
		name  string
		args  []string
		param string
	}{
		{
			name: "Octal",
			args: []string{"--octalParam=923"},
		},
		{
			name: "URL",
			args: []string{"--urlParam=a_b://abc"},
		},
		{
			name: "LogSeverity",
			args: []string{"--logSeverityParam=abc"},
		},
		{
			name: "Protocol",
			args: []string{"--protocolParam=pqr"},
		},
	}
	for _, k := range tests {
		t.Run(k.name, func(t *testing.T) {
			fs := declareFlags()
			v := bindFlags(fs)
			c := TestConfig{}
			args := []string{"test"}
			args = append(args, k.args...)
			err := fs.Parse(args)
			if err != nil {
				t.Fatalf("Flag parsing failed: %v", err)
			}

			err = v.Unmarshal(&c, viper.DecodeHook(DecodeHook()))

			assert.NotNil(t, err)
		})
	}
}
