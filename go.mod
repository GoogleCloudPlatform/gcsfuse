module github.com/googlecloudplatform/gcsfuse

go 1.21

require (
	cloud.google.com/go/compute/metadata v0.2.3
	cloud.google.com/go/storage v1.31.0
	contrib.go.opencensus.io/exporter/ocagent v0.7.0
	contrib.go.opencensus.io/exporter/stackdriver v0.13.11
	github.com/fsouza/fake-gcs-server v1.40.3
	github.com/googleapis/gax-go/v2 v2.11.0
	github.com/jacobsa/daemonize v0.0.0-20160101105449-e460293e890f
	github.com/jacobsa/fuse v0.0.0-20231003132804-d0f3daf365c3
	github.com/jacobsa/oglematchers v0.0.0-20150720000706-141901ea67cd
	github.com/jacobsa/oglemock v0.0.0-20150831005832-e94d794d06ff
	github.com/jacobsa/ogletest v0.0.0-20170503003838-80d50a735a11
	github.com/jacobsa/reqtrace v0.0.0-20150505043853-245c9e0234cb
	github.com/jacobsa/syncutil v0.0.0-20180201203307-228ac8e5a6c3
	github.com/jacobsa/timeutil v0.0.0-20170205232429-577e5acbbcf6
	github.com/jacobsa/util v0.0.0-20150810040848-976a6f4de67e
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0
	github.com/urfave/cli v1.22.5
	go.opencensus.io v0.24.0
	golang.org/x/net v0.10.0
	golang.org/x/oauth2 v0.8.0
	golang.org/x/sync v0.2.0
	golang.org/x/sys v0.8.0
	golang.org/x/text v0.9.0
	golang.org/x/time v0.3.0
	google.golang.org/api v0.126.0
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloud.google.com/go v0.110.2 // indirect
	cloud.google.com/go/compute v1.19.3 // indirect
	cloud.google.com/go/iam v1.1.0 // indirect
	cloud.google.com/go/monitoring v1.13.0 // indirect
	cloud.google.com/go/pubsub v1.30.0 // indirect
	cloud.google.com/go/trace v1.9.0 // indirect
	github.com/alexbrainman/goissue34681 v0.0.0-20191006012335-3fc7a47baff5 // indirect
	github.com/aws/aws-sdk-go v1.44.217 // indirect
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.1 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/s2a-go v0.1.4 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.3 // indirect
	github.com/gorilla/handlers v1.5.1 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.15.2 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/pkg/xattr v0.4.9 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/stretchr/testify v1.8.2 // indirect
	golang.org/x/crypto v0.9.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230530153820-e85fd2cbaebc // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230530153820-e85fd2cbaebc // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230530153820-e85fd2cbaebc // indirect
	google.golang.org/grpc v1.55.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)

// Retracted un-release version
retract v1.0.0
