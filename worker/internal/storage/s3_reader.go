package storage

import (
	"bytes"
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Reader struct {
	client *s3.Client
	bucket string
}

// NewS3Reader talks to any S3-compatible object store. Pass a non-empty
// endpoint to target a non-AWS provider (MinIO/R2/etc); pass "" for AWS
// S3 directly — same convention as catalog-writer-api.
func NewS3Reader(bucket, endpoint string) (*S3Reader, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		}
	})
	return &S3Reader{client: client, bucket: bucket}, nil
}

func (r *S3Reader) ReadObject(ctx context.Context, key string) ([]byte, error) {
	out, err := r.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &r.bucket, Key: &key})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, out.Body); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
