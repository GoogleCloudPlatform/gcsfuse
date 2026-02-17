module github.com/googlecloudplatform/gcsfuse/v3

go 1.24.11

require (
	cloud.google.com/go/auth v0.18.1
	cloud.google.com/go/auth/oauth2adapt v0.2.8
	cloud.google.com/go/compute/metadata v0.9.0
	cloud.google.com/go/iam v1.5.3
	cloud.google.com/go/profiler v0.4.3
	cloud.google.com/go/secretmanager v1.16.0
	cloud.google.com/go/storage v1.59.2
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.55.0
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v1.31.0
	github.com/fsouza/fake-gcs-server v1.52.2
	github.com/go-viper/mapstructure/v2 v2.5.0
	github.com/google/uuid v1.6.0
	github.com/googleapis/gax-go/v2 v2.16.0
	github.com/jacobsa/daemonize v0.0.0-20240917082746-f35568b6c3ec
	github.com/jacobsa/fuse v0.0.0-20251201175411-4b5f1a867296
	github.com/jacobsa/oglematchers v0.0.0-20150720000706-141901ea67cd
	github.com/jacobsa/oglemock v0.0.0-20150831005832-e94d794d06ff
	github.com/jacobsa/ogletest v0.0.0-20170503003838-80d50a735a11
	github.com/jacobsa/reqtrace v0.0.0-20150505043853-245c9e0234cb
	github.com/jacobsa/syncutil v0.0.0-20180201203307-228ac8e5a6c3
	github.com/jacobsa/timeutil v0.0.0-20170205232429-577e5acbbcf6
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0
	github.com/ncruces/go-dns v1.3.2
	github.com/pkg/xattr v0.4.12
	github.com/prometheus/client_golang v1.23.2
	github.com/prometheus/client_model v0.6.2
	github.com/prometheus/common v0.67.5
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
	github.com/spf13/viper v1.21.0
	github.com/stretchr/testify v1.11.1
	go.opencensus.io v0.24.0
	go.opentelemetry.io/contrib/detectors/gcp v1.40.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.65.0
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace v0.65.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.65.0
	go.opentelemetry.io/otel v1.40.0
	go.opentelemetry.io/otel/exporters/prometheus v0.62.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.40.0
	go.opentelemetry.io/otel/metric v1.40.0
	go.opentelemetry.io/otel/sdk v1.40.0
	go.opentelemetry.io/otel/sdk/metric v1.40.0
	go.opentelemetry.io/otel/trace v1.40.0
	golang.org/x/net v0.49.0
	golang.org/x/oauth2 v0.34.0
	golang.org/x/sync v0.19.0
	golang.org/x/sys v0.40.0
	golang.org/x/text v0.33.0
	golang.org/x/time v0.14.0
	google.golang.org/api v0.264.0
	google.golang.org/grpc v1.79.1
	google.golang.org/protobuf v1.36.11
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cel.dev/expr v0.25.1 // indirect
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/longrunning v0.7.0 // indirect
	cloud.google.com/go/monitoring v1.24.3 // indirect
	cloud.google.com/go/pubsub v1.50.1 // indirect
	cloud.google.com/go/pubsub/v2 v2.3.0 // indirect
	cloud.google.com/go/trace v1.11.7 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.31.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.55.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cncf/xds/go v0.0.0-20251210132809-ee656c7534f5 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.36.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.3.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-jose/go-jose/v4 v4.1.3 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/google/pprof v0.0.0-20251213031049-b05bdaca462f // indirect
	github.com/google/renameio/v2 v2.0.1 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.11 // indirect
	github.com/gorilla/handlers v1.5.2 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/otlptranslator v1.0.0 // indirect
	github.com/prometheus/procfs v0.19.2 // indirect
	github.com/sagikazarmark/locafero v0.12.0 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spiffe/go-spiffe/v2 v2.6.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	google.golang.org/genproto v0.0.0-20251222181119-0a764e51fe1b // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251222181119-0a764e51fe1b // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260128011058-8636f8732409 // indirect
)
