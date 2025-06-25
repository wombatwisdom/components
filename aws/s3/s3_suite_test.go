package s3_test

import (
	"context"
	"crypto/tls"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/spec"
	"github.com/wombatwisdom/components/test"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"
)

var env spec.Environment
var server *httptest.Server
var awsCfg aws.Config
var s3Client *s3.Client

func TestMqtt(t *testing.T) {
	RegisterFailHandler(Fail)

	BeforeSuite(func() {
		backend := s3mem.New()
		faker := gofakes3.New(backend, gofakes3.WithHostBucket(true))
		server = httptest.NewServer(faker.Server())

		awsCfg, _ = config.LoadDefaultConfig(
			context.TODO(),
			config.WithRegion("us-east-1"),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("KEY", "SECRET", "SESSION")),
			config.WithHTTPClient(&http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
					// Override the dial address because the SDK uses the bucket name as a subdomain.
					DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
						dialer := net.Dialer{
							Timeout:   30 * time.Second,
							KeepAlive: 30 * time.Second,
						}
						s3URL, _ := url.Parse(server.URL)
						return dialer.DialContext(ctx, network, s3URL.Host)
					},
				},
			}),
		)

		s3Client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.UsePathStyle = true
		})

		os.Setenv("AWS_ENDPOINT_URL_S3", server.URL)

		env = test.TestEnvironment()
	})

	AfterSuite(func() {
		if server != nil {
			server.Close()
		}
	})

	RunSpecs(t, "S3 Suite")
}
