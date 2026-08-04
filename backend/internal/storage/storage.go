// Package storage uploads guest-submitted images and audio to S3 (or any
// S3-compatible provider — Cloudflare R2, Backblaze B2, MinIO for local dev)
// and returns a public URL to store on the entry record.
package storage

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/yourorg/autograph-backend/internal/config"
)

type Storage struct {
	client   *s3.Client
	uploader *manager.Uploader
	bucket   string
	// publicBaseURL is prefixed onto uploaded object keys to build the URL
	// stored on entries. For AWS S3 this is the bucket's public/CDN URL;
	// for R2/MinIO set it to your public bucket domain.
	publicBaseURL string
}

func New(ctx context.Context, cfg config.S3Config, publicBaseURL string) (*Storage, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("loading aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.UsePathStyle
	})

	return &Storage{
		client:        client,
		uploader:      manager.NewUploader(client),
		bucket:        cfg.Bucket,
		publicBaseURL: publicBaseURL,
	}, nil
}

// UploadFile reads a multipart file (from an image/audio form field), uploads
// it under a random key namespaced by category, and returns its public URL.
func (s *Storage) UploadFile(ctx context.Context, categoryID string, fh *multipart.FileHeader) (string, error) {
	file, err := fh.Open()
	if err != nil {
		return "", fmt.Errorf("opening upload: %w", err)
	}
	defer file.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(file); err != nil {
		return "", fmt.Errorf("reading upload: %w", err)
	}

	key := fmt.Sprintf("entries/%s/%s-%s", categoryID, uuid.NewString(), fh.Filename)

	_, err = s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String(fh.Header.Get("Content-Type")),
	})
	if err != nil {
		return "", fmt.Errorf("uploading to s3: %w", err)
	}

	return fmt.Sprintf("%s/%s", s.publicBaseURL, key), nil
}

// UploadBytes uploads raw bytes (e.g. a generated PDF) under an exact key —
// used by the export worker, which builds a file in memory rather than
// receiving it as a multipart upload.
func (s *Storage) UploadBytes(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("uploading to s3: %w", err)
	}
	return fmt.Sprintf("%s/%s", s.publicBaseURL, key), nil
}
