module github.com/wombatwisdom/components/nats

go 1.24.1

toolchain go1.24.4

require (
	github.com/nats-io/jwt/v2 v2.7.4
	github.com/nats-io/nats-server/v2 v2.11.4
	github.com/nats-io/nats.go v1.43.0
	github.com/nats-io/nkeys v0.4.11
	github.com/onsi/ginkgo/v2 v2.23.4
	github.com/onsi/gomega v1.37.0
	github.com/wombatwisdom/components/framework v0.0.0-00010101000000-000000000000
)

require (
	github.com/expr-lang/expr v1.17.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.3.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/go-tpm v0.9.5 // indirect
	github.com/google/pprof v0.0.0-20250403155104-27863c87afa6 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/minio/highwayhash v1.0.3 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	golang.org/x/crypto v0.38.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	golang.org/x/tools v0.33.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/wombatwisdom/components/framework => ../../framework
