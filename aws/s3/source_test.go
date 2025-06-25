package s3_test

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	as3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wombatwisdom/components/aws/s3"
	"github.com/wombatwisdom/components/test"
	"strings"
)

var _ = Describe("Source", func() {
	var src *s3.Source

	BeforeEach(func() {
		var err error
		src, err = s3.NewSource(env, s3.SourceConfig{
			Config: awsCfg,
			Bucket: "test",
			Prefix: "test/files/",
		})
		Expect(err).ToNot(HaveOccurred())

		err = src.Connect(context.Background())
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		src.Disconnect(context.Background())
	})

	When("Reading a file from S3", func() {
		var bucket string

		BeforeEach(func() {
			bucket = fmt.Sprintf("test-%s", uuid.New().String())
			_, err := s3Client.CreateBucket(context.Background(), &as3.CreateBucketInput{
				Bucket: aws.String(bucket),
			})
			Expect(err).ToNot(HaveOccurred())

			// create the file in S3
			key := fmt.Sprintf("test/files/%s", uuid.New().String())
			_, err = s3Client.PutObject(context.Background(), &as3.PutObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
				Body:   strings.NewReader("hello, world"),
			})
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			// delete the s3 bucket
			_, err := s3Client.DeleteBucket(context.Background(), &as3.DeleteBucketInput{
				Bucket: aws.String(bucket),
			})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should receive a single message in the source", func() {
			collector := test.NewListCollector()
			defer collector.Disconnect()

			err := src.Read(context.Background(), collector)
			Expect(err).ToNot(HaveOccurred())

			Expect(collector.Messages()).To(HaveLen(1))
		})
	})
})
