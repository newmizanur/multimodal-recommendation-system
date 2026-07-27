package storage

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Signer struct {
	presign *s3.PresignClient
	bucket  string
}

// NewS3Signer talks to any S3-compatible object store — AWS S3, MinIO,
// Cloudflare R2, Backblaze B2, etc. Pass a non-empty endpoint to target
// a non-AWS provider (e.g. http://minio:9000 locally); pass "" to use
// AWS S3 directly.
func NewS3Signer(bucket, endpoint string) (*S3Signer, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true // required by MinIO and most non-AWS S3-compatible stores
		}
	})
	return &S3Signer{presign: s3.NewPresignClient(client), bucket: bucket}, nil
}

// PresignPut returns a short-lived URL the tenant's client uploads
// directly to — image bytes never transit through this API.
func (s *S3Signer) PresignPut(ctx context.Context, key string) (string, error) {
	req, err := s.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	}, s3.WithPresignExpires(10*time.Minute))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}
