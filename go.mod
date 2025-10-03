module github.com/wombatwisdom/components

go 1.25.1

require (
	// AWS dependencies
	github.com/aws/aws-sdk-go-v2 v1.36.5
	github.com/aws/aws-sdk-go-v2/config v1.29.17
	github.com/aws/aws-sdk-go-v2/credentials v1.17.70
	github.com/aws/aws-sdk-go-v2/service/eventbridge v1.36.7
	github.com/aws/aws-sdk-go-v2/service/pipes v1.19.5
	github.com/aws/aws-sdk-go-v2/service/s3 v1.81.0
	github.com/aws/aws-sdk-go-v2/service/sqs v1.38.8

	// MQTT dependencies
	github.com/eclipse/paho.mqtt.golang v1.5.0

	// Framework dependencies (now unified)
	github.com/expr-lang/expr v1.17.4
	github.com/go-viper/mapstructure/v2 v2.3.0

	// Utilities
	github.com/google/uuid v1.6.0

	// IBM MQ dependencies
	github.com/ibm-messaging/mq-golang/v5 v5.6.2
	github.com/johannesboyne/gofakes3 v0.0.0-20250106100439-5c39aecd6999
	github.com/mochi-mqtt/server/v2 v2.7.9

	// NATS dependencies
	github.com/nats-io/jwt/v2 v2.8.0
	github.com/nats-io/nats-server/v2 v2.12.0
	github.com/nats-io/nats.go v1.45.0
	github.com/nats-io/nkeys v0.4.11

	// Testing dependencies
	github.com/onsi/ginkgo/v2 v2.23.4
	github.com/onsi/gomega v1.37.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/Jeffail/gabs/v2 v2.7.0
	github.com/OneOfOne/xxhash v1.2.8
	github.com/gofrs/uuid/v5 v5.3.2
	github.com/matoous/go-nanoid/v2 v2.1.0
	github.com/redpanda-data/benthos/v4 v4.57.1
	github.com/segmentio/ksuid v1.0.4
	github.com/stretchr/testify v1.10.0
	github.com/tilinna/z85 v1.0.0
	github.com/xeipuuv/gojsonschema v1.2.0
	go.opentelemetry.io/otel v1.37.0
	go.opentelemetry.io/otel/trace v1.37.0
	golang.org/x/text v0.29.0
)

require (
	cuelang.org/go v0.13.2 // indirect
	github.com/Jeffail/shutdown v1.0.0 // indirect
	github.com/antithesishq/antithesis-sdk-go v0.4.3-default-no-op // indirect
	github.com/aws/aws-sdk-go v1.44.256 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.11 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.32 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.36 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.36 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.36 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.7.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.25.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.30.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.34.0 // indirect
	github.com/aws/smithy-go v1.22.4 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cockroachdb/apd/v3 v3.2.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/go-tpm v0.9.5 // indirect
	github.com/google/pprof v0.0.0-20250403155104-27863c87afa6 // indirect
	github.com/gorilla/handlers v1.5.2 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/minio/highwayhash v1.0.3 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/nsf/jsondiff v0.0.0-20210926074059-1e845ec5d249 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/rs/xid v1.4.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/ryszard/goskiplist v0.0.0-20150312221310-2dfbae5fcf46 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/urfave/cli/v2 v2.27.7 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.shabbyrobe.org/gocovmerge v0.0.0-20230507111327-fa4f82cfbf4d // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	golang.org/x/crypto v0.42.0 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/time v0.13.0 // indirect
	golang.org/x/tools v0.36.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
)
