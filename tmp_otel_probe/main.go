// Temporary debugging tool. NOT meant to be committed.
// Prints every OTel resource attribute gcsfuse can actually detect in the
// current environment, plus the raw GCE/GKE metadata-server values behind them.
//
// Run:  go run ./tmp_otel_probe
// Delete when done: rm -rf tmp_otel_probe
package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"cloud.google.com/go/compute/metadata"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Mirrors internal/monitor/otelexporters.go:getResource().
func detectResource(ctx context.Context, mountID string) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithDetectors(gcp.NewDetector()),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceName("gcsfuse"),
			semconv.ServiceVersion("probe"),
			semconv.ServiceInstanceID(mountID),
		),
	)
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("=== 1. Environment ===")
	fmt.Printf("metadata.OnGCE()            : %v\n", metadata.OnGCE())
	for _, e := range []string{
		"KUBERNETES_SERVICE_HOST", // GKE detection gate
		"HOSTNAME",                // usually the pod name
		"NAMESPACE", "POD_NAME", "POD_NAMESPACE", "CONTAINER_NAME", "NODE_NAME",
		"GOOGLE_CLOUD_PROJECT",
		"OTEL_RESOURCE_ATTRIBUTES", // NOTE: gcsfuse does NOT read this today.
	} {
		fmt.Printf("env %-26s: %q\n", e, os.Getenv(e))
	}

	fmt.Println("\n=== 2. Raw metadata server values (what the detector reads) ===")
	type probe struct {
		label string
		fn    func() (string, error)
	}
	probes := []probe{
		{"project/project-id", func() (string, error) { return metadata.ProjectIDWithContext(ctx) }},
		{"project/numeric-project-id", func() (string, error) { return metadata.NumericProjectIDWithContext(ctx) }},
		{"instance/id", func() (string, error) { return metadata.InstanceIDWithContext(ctx) }},
		{"instance/name", func() (string, error) { return metadata.InstanceNameWithContext(ctx) }},
		{"instance/hostname", func() (string, error) { return metadata.HostnameWithContext(ctx) }},
		{"instance/zone", func() (string, error) { return metadata.ZoneWithContext(ctx) }},
		{"instance/machine-type", func() (string, error) { return metadata.GetWithContext(ctx, "instance/machine-type") }},
		{"attr cluster-name", func() (string, error) { return metadata.InstanceAttributeValueWithContext(ctx, "cluster-name") }},
		{"attr cluster-location", func() (string, error) { return metadata.InstanceAttributeValueWithContext(ctx, "cluster-location") }},
		{"attr cluster-uid", func() (string, error) { return metadata.InstanceAttributeValueWithContext(ctx, "cluster-uid") }},
	}
	for _, p := range probes {
		v, err := p.fn()
		if err != nil {
			fmt.Printf("%-28s: UNAVAILABLE (%v)\n", p.label, err)
			continue
		}
		fmt.Printf("%-28s: %s\n", p.label, v)
	}

	fmt.Println("\n=== 3. OTel resource attributes gcsfuse would attach ===")
	res, err := detectResource(ctx, "probe-mount-id")
	if err != nil {
		fmt.Printf("detection returned error (partial results may still follow): %v\n", err)
	}
	if res == nil {
		fmt.Println("no resource detected")
		return
	}
	attrs := res.Attributes()
	sort.Slice(attrs, func(i, j int) bool { return attrs[i].Key < attrs[j].Key })
	for _, a := range attrs {
		fmt.Printf("%-40s = %s\n", string(a.Key), a.Value.Emit())
	}
	fmt.Printf("\nschema URL: %s\ntotal attributes: %d\n", res.SchemaURL(), len(attrs))
}
