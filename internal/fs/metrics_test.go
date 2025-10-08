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

package fs_test

import (
	"context"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/wrappers"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func createTestFileSystemWithMetrics(ctx context.Context, t *testing.T) (gcs.Bucket, fuseutil.FileSystem, metrics.MetricHandle, *metric.ManualReader) {
	t.Helper()
	origProvider := otel.GetMeterProvider()
	t.Cleanup(func() { otel.SetMeterProvider(origProvider) })
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)

	mh, err := metrics.NewOTelMetrics(ctx, 1, 100)
	require.NoError(t, err, "metrics.NewOTelMetrics")
	bucketName := "test-bucket"
	bucket := fake.NewFakeBucket(timeutil.RealClock(), bucketName, gcs.BucketType{Hierarchical: false})
	serverCfg := &fs.ServerConfig{
		NewConfig: &cfg.Config{
			Write: cfg.WriteConfig{
				GlobalMaxBlocks: 1,
			},
			Read: cfg.ReadConfig{
				GlobalMaxBlocks: 1,
			},
		},
		MetricHandle: mh,
		CacheClock:   &timeutil.SimulatedClock{},
		BucketName:   bucketName,
		BucketManager: &fakeBucketManager{
			buckets: map[string]gcs.Bucket{
				bucketName: bucket,
			},
		},
	}
	server, err := fs.NewFileSystem(ctx, serverCfg)
	require.NoError(t, err, "NewFileSystem")
	return bucket, server, mh, reader
}

func createWithContents(ctx context.Context, t *testing.T, bucket gcs.Bucket, name string, contents string) {
	err := storageutil.CreateObjects(ctx, bucket, map[string][]byte{name: []byte(contents)})
	require.NoError(t, err, "CreateObjects")
}

// verifyCounterMetric finds a counter metric and verifies that the data point
// matching the provided attributes has the expected value.
func verifyCounterMetric(t *testing.T, ctx context.Context, reader *metric.ManualReader, metricName string, attrs attribute.Set, expectedValue int64) {
	t.Helper()
	var rm metricdata.ResourceMetrics
	err := reader.Collect(ctx, &rm)
	require.NoError(t, err, "reader.Collect")
	encoder := attribute.DefaultEncoder()
	expectedKey := attrs.Encoded(encoder)

	require.Len(t, rm.ScopeMetrics, 1, "expected 1 scope metric")
	require.NotEmpty(t, rm.ScopeMetrics[0].Metrics, "expected at least 1 metric")

	foundMetric := false
	for _, m := range rm.ScopeMetrics[0].Metrics {
		if m.Name == metricName {
			foundMetric = true
			data, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok, "metric %s is not a Sum[int64], but %T", metricName, m.Data)

			foundDataPoint := false
			for _, dp := range data.DataPoints {
				if dp.Attributes.Encoded(encoder) == expectedKey {
					foundDataPoint = true
					assert.Equal(t, expectedValue, dp.Value, "metric value mismatch for attributes: %s", attrs.Encoded(encoder))
					break
				}
			}

			require.True(t, foundDataPoint, "Data point for attributes %v not found in %s metric", attrs, metricName)
			break
		}
	}

	require.True(t, foundMetric, "metric %s not found", metricName)
}

func TestLookUpInode_Metrics(t *testing.T) {
	testCases := []struct {
		name          string
		fileName      string
		createFile    bool
		expectedError error
	}{
		{
			name:          "non-existent file",
			fileName:      "non_existent",
			createFile:    false,
			expectedError: fuse.ENOENT,
		},
		{
			name:          "existing file",
			fileName:      "test",
			createFile:    true,
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			bucket, server, mh, reader := createTestFileSystemWithMetrics(ctx, t)
			const content = "test"
			if tc.createFile {
				createWithContents(ctx, t, bucket, tc.fileName, content)
			}

			server = wrappers.WithMonitoring(server, mh)
			op := &fuseops.LookUpInodeOp{
				Parent: fuseops.RootInodeID,
				Name:   tc.fileName,
			}

			err := server.LookUpInode(ctx, op)

			assert.Equal(t, tc.expectedError, err)
			attrs := attribute.NewSet(attribute.String("fs_op", "LookUpInode"))
			verifyCounterMetric(t, ctx, reader, "fs/ops_count", attrs, 1)
		})
	}
}
