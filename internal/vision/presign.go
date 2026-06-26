// Package vision — presign.go generates pre-signed URLs for downloading clips
// from Backblaze B2 via the S3-compatible API.
package vision

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// B2BucketName is the configured B2 bucket name for clip storage.
var B2BucketName = "watchdog-clips"

// SetB2Bucket configures the B2 bucket name for presign URL generation.
func SetB2Bucket(b string) { B2BucketName = b }

// presignerConfig holds the B2/S3 credentials for URL signing.
type presignerConfig struct {
	keyID    string
	appKey   string
	endpoint string
	bucket   string
}

// globalPresignerConfig is set at startup via SetPresignerCredentials.
var globalPresignerConfig *presignerConfig

// SetPresignerCredentials configures the B2 credentials for presigning.
// Call this at startup after loading config.
func SetPresignerCredentials(keyID, appKey, bucket, endpoint string) {
	globalPresignerConfig = &presignerConfig{
		keyID:    keyID,
		appKey:   appKey,
		endpoint: endpoint,
		bucket:   bucket,
	}
	if bucket != "" {
		B2BucketName = bucket
	}
}

// b2Endpoint returns the S3-compatible endpoint URL for the bucket.
// B2 S3-compatible URLs follow the pattern: https://s3.<region>.backblazeb2.com
// The region is embedded in the endpoint, e.g. s3.us-west-004.backblazeb2.com.
// If no endpoint is configured, defaults to the common s3.us-west-004.
func b2Endpoint() string {
	if globalPresignerConfig != nil && globalPresignerConfig.endpoint != "" {
		return globalPresignerConfig.endpoint
	}
	return "https://s3.us-west-004.backblazeb2.com"
}

// GeneratePresignURL returns a cryptographically signed, time-limited URL
// for downloading a clip from B2 via the S3-compatible API.
// The URL expires after expiryMinutes (default 15, max 60 for B2 free tier).
// Without valid B2 credentials configured, falls back to an unsigned URL
// with a clear warning in the URL (for dev only — never use in production).
func (s *Service) GeneratePresignURL(ctx context.Context, clip Clip, expiryMinutes int) (string, error) {
	if clip.S3Key == "" {
		return "", fmt.Errorf("vision: clip has no S3 key")
	}
	if expiryMinutes <= 0 {
		expiryMinutes = 60
	}
	if expiryMinutes > 60*24 { // B2 allows up to 7 days but cap at 24h
		expiryMinutes = 60 * 24
	}

	bucket := clip.S3Bucket
	if bucket == "" {
		bucket = B2BucketName
	}

	// If no B2 credentials configured, return unsigned URL (dev mode only)
	if globalPresignerConfig == nil || globalPresignerConfig.keyID == "" {
		expires := time.Now().Add(time.Duration(expiryMinutes) * time.Minute).Unix()
		return fmt.Sprintf("https://f000.backblazeb2.com/file/%s/%s?expires=%d&warning=unsigned_dev_only", bucket, clip.S3Key, expires), nil
	}

	// Build S3 client configured for B2
	cfg, err := awscfg.LoadDefaultConfig(ctx,
		awscfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			globalPresignerConfig.keyID,
			globalPresignerConfig.appKey,
			"",
		)),
		awscfg.WithRegion("us-west-004"),
	)
	if err != nil {
		return "", fmt.Errorf("vision: load s3 config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(b2Endpoint())
		o.UsePathStyle = true // B2 requires path-style addressing
	})

	// Create a presign client and generate the signed URL
	presignClient := s3.NewPresignClient(s3Client)
	presignReq, err := presignClient.PresignGetObject(ctx,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(clip.S3Key),
		},
		s3.WithPresignExpires(time.Duration(expiryMinutes)*time.Minute),
	)
	if err != nil {
		return "", fmt.Errorf("vision: presign url: %w", err)
	}

	return presignReq.URL, nil
}
