module github.com/wombatwisdom/components/eventbridge

go 1.24.1

toolchain go1.24.4

require (
	github.com/aws/aws-sdk-go-v2 v1.36.5
	github.com/aws/aws-sdk-go-v2/service/eventbridge v1.36.7
	github.com/aws/aws-sdk-go-v2/service/pipes v1.19.5
	github.com/aws/aws-sdk-go-v2/service/sqs v1.38.8
	github.com/onsi/ginkgo/v2 v2.23.4
	github.com/onsi/gomega v1.37.0
	github.com/wombatwisdom/components/framework v0.0.0-00010101000000-000000000000
)

require (
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.36 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.36 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.36 // indirect
	github.com/aws/smithy-go v1.22.4 // indirect
	github.com/expr-lang/expr v1.17.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.3.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20250403155104-27863c87afa6 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	golang.org/x/tools v0.33.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/wombatwisdom/components/framework => ../framework
