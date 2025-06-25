package s3_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	as3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/aws-s3"
	"github.com/wombatwisdom/components/framework/test"
)

var _ = Describe("Input", func() {
	var input *s3.Input

	BeforeEach(func() {
		// Input will be created in the When block with the correct bucket
	})

	AfterEach(func() {
		if input != nil {
			_ = input.Disconnect(context.Background())
		}
	})

	When("Reading a file from S3", func() {
		var bucket string
		var key string

		BeforeEach(func() {
			bucket = "testbucket"
			_, err := s3Client.CreateBucket(context.Background(), &as3.CreateBucketInput{
				Bucket: aws.String(bucket),
			})
			Expect(err).ToNot(HaveOccurred())

			// create the file in S3
			key = fmt.Sprintf("test/files/%s", uuid.New().String())
			_, err = s3Client.PutObject(context.Background(), &as3.PutObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
				Body:   strings.NewReader("hello, world"),
			})
			Expect(err).ToNot(HaveOccurred())

			// Create input with the correct bucket and test endpoint
			testConfig := awsCfg.Copy()
			input, err = s3.NewInput(env, s3.InputConfig{
				Config:             testConfig,
				Bucket:             bucket,
				Prefix:             "test/files/",
				ForcePathStyleURLs: true,
				EndpointURL:        aws.String(server.URL),
			})
			Expect(err).ToNot(HaveOccurred())

			err = input.Connect(context.Background())
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			// Clean up objects in bucket (fake S3 doesn't require explicit bucket deletion)
			_, _ = s3Client.DeleteObject(context.Background(), &as3.DeleteObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
			})
		})

		It("should receive a single message in the source", func() {
			collector := test.NewListCollector()
			defer func() { _ = collector.Disconnect() }()

			err := input.Read(context.Background(), collector)
			Expect(err).ToNot(HaveOccurred())

			Expect(collector.Messages()).To(HaveLen(1))
		})
	})
})
