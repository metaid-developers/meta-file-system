package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Storage AWS S3 compatible storage (supports AWS S3 and MinIO)
type S3Storage struct {
	client *s3.Client
	bucket string
}

// NewS3Storage create S3 storage instance
func NewS3Storage(region, endpoint, accessKey, secretKey, bucketName string) (*S3Storage, error) {
	if accessKey == "" || secretKey == "" || bucketName == "" {
		return nil, ErrInvalid
	}

	ctx := context.Background()

	// Create credentials
	creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")

	// Load AWS configuration
	var cfg aws.Config
	var err error

	if endpoint != "" {
		// Custom endpoint (for MinIO or S3-compatible services)
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(creds),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}

		// Create S3 client with custom endpoint
		client := s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true // Required for MinIO
		})

		return &S3Storage{
			client: client,
			bucket: bucketName,
		}, nil
	} else {
		// Standard AWS S3
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(creds),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}

		client := s3.NewFromConfig(cfg)

		return &S3Storage{
			client: client,
			bucket: bucketName,
		}, nil
	}
}

// NewMinIOStorage create MinIO storage instance (alias for S3Storage)
func NewMinIOStorage(endpoint, accessKey, secretKey, bucketName string) (*S3Storage, error) {
	// MinIO uses "us-east-1" as default region, but it doesn't really matter
	return NewS3Storage("us-east-1", endpoint, accessKey, secretKey, bucketName)
}

// Save save file to S3
func (s *S3Storage) Save(key string, data []byte) error {
	ctx := context.Background()

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to upload to s3: %w", err)
	}

	return nil
}

// Get get file from S3
func (s *S3Storage) Get(key string) ([]byte, error) {
	ctx := context.Background()

	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get from s3: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read s3 object: %w", err)
	}

	return data, nil
}

// Delete delete file from S3
func (s *S3Storage) Delete(key string) error {
	ctx := context.Background()

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from s3: %w", err)
	}

	return nil
}

// Exists check if file exists in S3
func (s *S3Storage) Exists(key string) bool {
	ctx := context.Background()

	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	return err == nil
}

// InitiateMultipartUpload initiate multipart upload
func (s *S3Storage) InitiateMultipartUpload(key string) (string, error) {
	ctx := context.Background()

	result, err := s.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", fmt.Errorf("failed to initiate multipart upload: %w", err)
	}

	return *result.UploadId, nil
}

// UploadPart upload a part
func (s *S3Storage) UploadPart(key, uploadId string, partNumber int, data []byte) (string, error) {
	ctx := context.Background()

	result, err := s.client.UploadPart(ctx, &s3.UploadPartInput{
		Bucket:     aws.String(s.bucket),
		Key:        aws.String(key),
		UploadId:   aws.String(uploadId),
		PartNumber: aws.Int32(int32(partNumber)),
		Body:       bytes.NewReader(data),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload part %d: %w", partNumber, err)
	}

	return *result.ETag, nil
}

// CompleteMultipartUpload complete multipart upload
func (s *S3Storage) CompleteMultipartUpload(key, uploadId string, parts []PartInfo) error {
	ctx := context.Background()

	// Convert PartInfo to S3 CompletedPart
	completedParts := make([]types.CompletedPart, 0, len(parts))
	for _, p := range parts {
		completedParts = append(completedParts, types.CompletedPart{
			PartNumber: aws.Int32(int32(p.PartNumber)),
			ETag:       aws.String(p.ETag),
		})
	}

	// Sort parts by part number
	sort.Slice(completedParts, func(i, j int) bool {
		return *completedParts[i].PartNumber < *completedParts[j].PartNumber
	})

	_, err := s.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadId),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	return nil
}

// AbortMultipartUpload abort multipart upload
func (s *S3Storage) AbortMultipartUpload(key, uploadId string) error {
	ctx := context.Background()

	_, err := s.client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadId),
	})
	if err != nil {
		return fmt.Errorf("failed to abort multipart upload: %w", err)
	}

	return nil
}

// ListParts list uploaded parts
func (s *S3Storage) ListParts(key, uploadId string) ([]PartInfo, error) {
	ctx := context.Background()

	result, err := s.client.ListParts(ctx, &s3.ListPartsInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadId),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list parts: %w", err)
	}

	parts := make([]PartInfo, 0, len(result.Parts))
	for _, p := range result.Parts {
		parts = append(parts, PartInfo{
			PartNumber: int(*p.PartNumber),
			ETag:       *p.ETag,
			Size:       *p.Size,
		})
	}

	return parts, nil
}

// GetMultipartUpload get complete file from multipart upload (after completion)
func (s *S3Storage) GetMultipartUpload(key, uploadId string) ([]byte, error) {
	// After multipart upload is completed, the file is stored at the key
	return s.Get(key)
}
