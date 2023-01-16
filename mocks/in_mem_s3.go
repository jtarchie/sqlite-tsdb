package mocks

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	fakes3 "github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
)

type S3Server struct {
	*httptest.Server
	*s3.Client

	bucketName string
}

func NewS3Server(
	bucketName string,
) (*S3Server, error) {
	backend := s3mem.New()
	faker := fakes3.New(backend)
	server := httptest.NewServer(faker.Server())

	client, err := news3Client(server.URL)
	if err != nil {
		return nil, fmt.Errorf("client error: %w", err)
	}

	err = setupS3Client(client, bucketName)
	if err != nil {
		return nil, fmt.Errorf("client setup error: %w", err)
	}

	return &S3Server{
		Server:     server,
		Client:     client,
		bucketName: bucketName,
	}, nil
}

func setupS3Client(
	client *s3.Client,
	bucketName string,
) error {
	_, err := client.CreateBucket(context.TODO(), &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint("fake-region"),
		},
	})
	if err != nil {
		return fmt.Errorf("could not create bucket: %w", err)
	}

	return nil
}

func news3Client(url string) (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				"KEY",
				"SECRET",
				"SESSION",
			),
		),
		config.WithHTTPClient(&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			},
		}),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(func(_, _ string, _ ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: url}, nil
			}),
		),
	)

	if err != nil {
		return nil, fmt.Errorf("could not create config for s3 client: %w", err)
	}

	// Create an Amazon S3 v2 client, important to use o.UsePathStyle
	// alternatively change local DNS settings, e.g., in /etc/hosts
	// to support requests to http://<bucketname>.127.0.0.1:32947/...
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	}), nil
}

// All functions below come from: https://docs.aws.amazon.com/code-library/latest/ug/go_2_s3_code_examples.html

func (s *S3Server) HasObject(name string) (int, error) {
	matcher, err := regexp.Compile(name)
	if err != nil {
		return 0, fmt.Errorf("could not compile regex for HasObject: %w", err)
	}

	count := 0

	contextTimeout, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	result, err := s.ListObjectsV2(
		contextTimeout,
		&s3.ListObjectsV2Input{
			Bucket: aws.String(s.bucketName),
		},
	)
	if err != nil {
		return count, fmt.Errorf("couldn't list objects in bucket %v: %w", s.bucketName, err)
	}

	for _, content := range result.Contents {
		if matcher.MatchString(*content.Key) {
			count++
		}
	}

	return count, nil
}

func (s *S3Server) PutObject(
	name string,
	contents io.ReadSeeker,
) error {
	contextTimeout, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := s.Client.PutObject(
		contextTimeout,
		&s3.PutObjectInput{
			Bucket: aws.String(s.bucketName),
			Key:    aws.String(name),
			Body:   contents,
		},
	)
	if err != nil {
		return fmt.Errorf("couldn't upload reader to %v as %v: %w", s.bucketName, name, err)
	}

	return nil
}

func (s *S3Server) Close() {
	s.Server.Close()
}
