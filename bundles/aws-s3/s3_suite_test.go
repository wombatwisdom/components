package s3_test

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/framework/spec"
	"github.com/wombatwisdom/components/framework/test"
)

var env spec.Environment
var server *httptest.Server
var awsCfg aws.Config
var s3Client *s3.Client

func TestMqtt(t *testing.T) {
	RegisterFailHandler(Fail)

	BeforeSuite(func() {
		backend := s3mem.New()
		faker := gofakes3.New(backend)
		server = httptest.NewServer(faker.Server())

		awsCfg, _ = config.LoadDefaultConfig(
			context.TODO(),
			config.WithRegion("us-east-1"),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("KEY", "SECRET", "SESSION")),
		)

		s3Client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.UsePathStyle = true
			o.BaseEndpoint = aws.String(server.URL)
		})

		_ = os.Setenv("AWS_ENDPOINT_URL_S3", server.URL)

		env = test.TestEnvironment()
	})

	AfterSuite(func() {
		if server != nil {
			server.Close()
		}
	})

	RunSpecs(t, "S3 Suite")
}
