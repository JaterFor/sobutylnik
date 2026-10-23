package profile

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// R2Storage issues presigned PUT URLs for uploading profile photos directly
// to Cloudflare R2 (S3-compatible), so the backend never buffers image bytes.
type R2Storage struct {
	client        *s3.PresignClient
	bucket        string
	publicBaseURL string
}

func NewR2Storage(accountID, accessKeyID, secretAccessKey, bucket, publicBaseURL string) *R2Storage {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	client := s3.New(s3.Options{
		Region:      "auto",
		BaseEndpoint: aws.String(endpoint),
		Credentials: credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
	})

	return &R2Storage{
		client:        s3.NewPresignClient(client),
		bucket:        bucket,
		publicBaseURL: publicBaseURL,
	}
}

// PresignUpload returns a short-lived presigned PUT URL and the object key
// the client must upload to, for the given user and content type.
func (s *R2Storage) PresignUpload(ctx context.Context, userID, contentType string) (uploadURL, objectKey string, err error) {
	objectKey = fmt.Sprintf("profiles/%s/%s", userID, uuid.NewString())

	req, err := s.client.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(objectKey),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(5*time.Minute))
	if err != nil {
		return "", "", err
	}

	return req.URL, objectKey, nil
}

func (s *R2Storage) PublicURL(objectKey string) string {
	return s.publicBaseURL + "/" + objectKey
}
