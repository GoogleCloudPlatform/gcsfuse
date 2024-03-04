module github.com/googlecloudplatform/gcsfuse

go 1.22.0

require (
	cloud.google.com/go/compute/metadata v0.2.3
	cloud.google.com/go/storage v1.38.0
	contrib.go.opencensus.io/exporter/ocagent v0.7.0
	contrib.go.opencensus.io/exporter/stackdriver v0.13.14
	github.com/fsouza/fake-gcs-server v1.47.8
	github.com/google/uuid v1.6.0
	github.com/googleapis/gax-go/v2 v2.12.1
	github.com/jacobsa/daemonize v0.0.0-20160101105449-e460293e890f
	github.com/jacobsa/fuse v0.0.0-20231003132804-d0f3daf365c3
	github.com/jacobsa/oglematchers v0.0.0-20150720000706-141901ea67cd
	github.com/jacobsa/oglemock v0.0.0-20150831005832-e94d794d06ff
	github.com/jacobsa/ogletest v0.0.0-20170503003838-80d50a735a11
	github.com/jacobsa/reqtrace v0.0.0-20150505043853-245c9e0234cb
	github.com/jacobsa/syncutil v0.0.0-20180201203307-228ac8e5a6c3
	github.com/jacobsa/timeutil v0.0.0-20170205232429-577e5acbbcf6
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0
	github.com/urfave/cli v1.22.14
	go.opencensus.io v0.24.0
	golang.org/x/net v0.21.0
	golang.org/x/oauth2 v0.17.0
	golang.org/x/sync v0.6.0
	golang.org/x/sys v0.17.0
	golang.org/x/text v0.14.0
	golang.org/x/time v0.5.0
	google.golang.org/api v0.167.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloud.google.com/go v0.112.0 // indirect
	cloud.google.com/go/compute v1.23.4 // indirect
	cloud.google.com/go/iam v1.1.6 // indirect
	cloud.google.com/go/monitoring v1.17.1 // indirect
	cloud.google.com/go/pubsub v1.36.1 // indirect
	cloud.google.com/go/trace v1.10.5 // indirect
	github.com/aws/aws-sdk-go v1.44.217 // indirect
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/renameio/v2 v2.0.0 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/gorilla/handlers v1.5.2 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.15.2 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/pkg/xattr v0.4.9 // indirect
	github.com/prometheus/prometheus v0.35.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.48.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.48.0 // indirect
	go.opentelemetry.io/otel v1.23.0 // indirect
	go.opentelemetry.io/otel/metric v1.23.0 // indirect
	go.opentelemetry.io/otel/trace v1.23.0 // indirect
	golang.org/x/crypto v0.19.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto v0.0.0-20240205150955-31a09d347014 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240205150955-31a09d347014 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240213162025-012b6fc9bca9 // indirect
	google.golang.org/grpc v1.61.1 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
)

// Retracted un-release version
retract v1.0.0
