// Package vision — presign.go generates pre-signed URLs for downloading clips
// from B2/S3-compatible storage.
package vision

import (
	"context"
	"fmt"
	"time"
)

// B2BucketName is the configured B2 bucket name for clip storage. Set via
// SetB2Bucket at startup; defaults to a placeholder.
var B2BucketName = "watchdog-clips"

// SetB2Bucket configures the B2 bucket name for presign URL generation.
func SetB2Bucket(b string) { B2BucketName = b }

// GeneratePresignURL returns a pre-signed URL for downloading the clip from
// B2/S3-compatible storage. This is a placeholder implementation that returns
// a deterministic URL; replace with actual B2/S3 SDK presigning when storage
// credentials are wired in.
func (s *Service) GeneratePresignURL(ctx context.Context, clip Clip, expiryMinutes int) (string, error) {
	if clip.S3Key == "" {
		return "", fmt.Errorf("vision: clip has no S3 key")
	}
	if expiryMinutes <= 0 {
		expiryMinutes = 60
	}
	bucket := clip.S3Bucket
	if bucket == "" {
		bucket = B2BucketName
	}
	expires := time.Now().Add(time.Duration(expiryMinutes) * time.Minute).Unix()
	return fmt.Sprintf("https://f000.backblazeb2.com/file/%s/%s?expires=%d", bucket, clip.S3Key, expires), nil
}