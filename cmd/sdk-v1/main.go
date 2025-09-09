package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/joho/godotenv"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

// GenerateImageKey generates a unique key for an image file
func GenerateImageKey(filename string) (string, error) {
	// Extract file extension
	parts := strings.Split(filename, ".")
	ext := "jpg" // default extension
	if len(parts) > 1 {
		ext = parts[len(parts)-1]
	}

	// Generate date-based prefix
	now := time.Now()
	yearMonth := now.Format("200601") // YYYYMM format

	// Generate nanoid for uniqueness
	nanoID, err := gonanoid.New(15)
	if err != nil {
		return "", fmt.Errorf("failed to generate nanoid: %w", err)
	}

	// Format: YYYYMM/nanoid.ext
	return fmt.Sprintf("%s/%s.%s", yearMonth, nanoID, ext), nil
}

// GenerateImageKeyWithEnv generates an image key with environment prefix for development
func GenerateImageKeyWithEnv(filename, environment string) (string, error) {
	key, err := GenerateImageKey(filename)
	if err != nil {
		return "", err
	}

	// Add dev prefix for development environment
	if environment == "dev" || environment == "development" {
		return "dev/" + key, nil
	}

	return key, nil
}

func main() {
	fmt.Println("Using AWS SDK v1 to avoid chunked encoding issues...")

	// Load environment variables from .env file
	err := godotenv.Load(".env")
	if err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
		log.Println("Falling back to system environment variables...")
	}
	// Get configuration from environment variables
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	region := os.Getenv("AWS_DEFAULT_REGION")
	bucketName := os.Getenv("AWS_BUCKET_NAME")
	endpointURL := os.Getenv("AWS_ENDPOINT_URL")
	environment := os.Getenv("ENV")

	// Validate required environment variables
	if accessKeyID == "" || secretAccessKey == "" || region == "" || bucketName == "" {
		log.Fatal("Missing required environment variables: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_DEFAULT_REGION, AWS_BUCKET_NAME")
	}

	fmt.Printf("AWS Config:\n")
	fmt.Printf("  Access Key ID: %s\n", accessKeyID)
	fmt.Printf("  Secret Access Key: %s (length: %d)\n", secretAccessKey[:5]+"***", len(secretAccessKey))
	fmt.Printf("  Region: %s\n", region)
	fmt.Printf("  Bucket: %s\n", bucketName)
	fmt.Printf("  Endpoint URL: %s\n", endpointURL)

	ctx := context.Background()

	// Initialize AWS SDK v1 session
	fmt.Println("\n--- Initializing AWS SDK v1 Client ---")
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			accessKeyID,
			secretAccessKey,
			"",
		),
		Endpoint:         aws.String(endpointURL),
		S3ForcePathStyle: aws.Bool(true),
		LogLevel:         aws.LogLevel(aws.LogDebugWithHTTPBody),
	})
	if err != nil {
		log.Fatalf("Failed to create AWS session: %v", err)
	}

	// Create S3 client
	s3Client := s3.New(sess)
	fmt.Printf("Using custom endpoint: %s\n", endpointURL)

	// Test 1: List buckets
	fmt.Println("\n--- Test 1: List Buckets ---")
	result, err := s3Client.ListBucketsWithContext(ctx, &s3.ListBucketsInput{})
	if err != nil {
		fmt.Printf("Error listing buckets: %v\n", err)
	} else {
		fmt.Printf("Successfully listed buckets: %d buckets found\n", len(result.Buckets))
		for _, bucket := range result.Buckets {
			fmt.Printf("  - %s\n", *bucket.Name)
		}
	}

	// Test 2: Check if specific bucket exists
	fmt.Println("\n--- Test 2: Head Bucket ---")
	_, err = s3Client.HeadBucketWithContext(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		fmt.Printf("Error checking bucket '%s': %v\n", bucketName, err)
	} else {
		fmt.Printf("Bucket '%s' exists and is accessible\n", bucketName)
	}

	// Test 3: Generate a unique key for file upload
	fmt.Println("\n--- Test 3: Generate File Key ---")
	filename := "test-upload.txt"
	key, err := GenerateImageKeyWithEnv(filename, environment)
	if err != nil {
		fmt.Printf("Error generating file key: %v\n", err)
		return
	}
	fmt.Printf("Generated file key: %s\n", key)

	// Test 4: Create and upload a test file
	fmt.Println("\n--- Test 4: Upload File ---")

	// File to upload
	fileContent := "Hello from AWS SDK v1!\nThis should work without chunked encoding."
	key = "test-folder/test-file.txt"
	bucket := "sharex"

	// Upload using AWS SDK v1 - this should not use chunked encoding
	_, err = s3Client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        strings.NewReader(fileContent),
		ContentType: aws.String("text/plain"),
	})
	if err != nil {
		log.Fatalf("upload failed: %v", err)
	}

	fmt.Printf("✓ File uploaded successfully with key: %s (%d bytes)\n", key, len(fileContent))

	// Test 5: Wait for object to exist and verify
	fmt.Println("\n--- Test 5: Verify Upload ---")
	err = s3Client.WaitUntilObjectExistsWithContext(
		ctx,
		&s3.HeadObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(key),
		},
	)
	if err != nil {
		fmt.Printf("Error waiting for object to exist: %v\n", err)
	} else {
		fmt.Printf("✓ Object exists and is accessible\n")
	}

	// Test 6: Get file metadata
	fmt.Println("\n--- Test 6: Get File Metadata ---")
	headResult, err := s3Client.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		fmt.Printf("Error getting file metadata: %v\n", err)
	} else {
		fmt.Printf("✓ File metadata retrieved:\n")
		fmt.Printf("  Content Length: %d bytes\n", *headResult.ContentLength)
		fmt.Printf("  Last Modified: %s\n", *headResult.LastModified)
		fmt.Printf("  ETag: %s\n", *headResult.ETag)
	}

	// Test 7: Generate public URL
	fmt.Println("\n--- Test 7: Generate Public URL ---")
	var publicURL string
	if endpointURL != "" {
		// Custom endpoint (like DigitalOcean Spaces, MinIO, etc.)
		endpointURL := strings.TrimSuffix(endpointURL, "/")
		publicURL = fmt.Sprintf("%s/%s/%s", endpointURL, bucketName, key)
	} else {
		// Standard AWS S3 URL
		publicURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, region, key)
	}
	fmt.Printf("✓ Public URL: %s\n", publicURL)

	// Test 8: Generate presigned URL
	fmt.Println("\n--- Test 8: Generate Presigned URL ---")
	req, _ := s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})
	presignedURL, err := req.Presign(15 * time.Minute) // 15 minutes expiry
	if err != nil {
		fmt.Printf("Error generating presigned URL: %v\n", err)
	} else {
		fmt.Printf("✓ Presigned URL: %s\n", presignedURL)
	}

	// Test 9: List files in bucket with prefix
	fmt.Println("\n--- Test 9: List Files ---")
	keyParts := strings.Split(key, "/")
	prefix := ""
	if len(keyParts) > 1 {
		prefix = keyParts[0] + "/" // Get the date prefix or env prefix
	}

	listResult, err := s3Client.ListObjectsV2WithContext(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucketName),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int64(10),
	})
	if err != nil {
		fmt.Printf("Error listing files: %v\n", err)
	} else {
		fmt.Printf("Found %d files with prefix '%s':\n", len(listResult.Contents), prefix)
		for i, obj := range listResult.Contents {
			fmt.Printf("  %d. %s (%d bytes, %s)\n", i+1, *obj.Key, *obj.Size, obj.LastModified.Format("2006-01-02 15:04:05"))
		}
	}

	// Test 10: Soft delete (copy to .deleted and remove original)
	fmt.Println("\n--- Test 10: Soft Delete ---")
	deletedKey := key + ".deleted"

	// Copy to deleted key
	_, err = s3Client.CopyObjectWithContext(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(bucketName),
		Key:        aws.String(deletedKey),
		CopySource: aws.String(fmt.Sprintf("%s/%s", bucketName, key)),
	})
	if err != nil {
		fmt.Printf("Error copying file for soft delete: %v\n", err)
	} else {
		fmt.Printf("✓ File copied to deleted key: %s\n", deletedKey)

		// Delete original
		_, err = s3Client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(key),
		})
		if err != nil {
			fmt.Printf("Error deleting original file: %v\n", err)
		} else {
			fmt.Printf("✓ Original file deleted\n")
		}
	}

	// Test 11: Verify soft delete
	fmt.Println("\n--- Test 11: Verify Soft Delete ---")

	// Check if original file is gone
	_, err = s3Client.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		fmt.Printf("✓ Original file no longer exists (expected)\n")
	} else {
		fmt.Printf("✗ Original file still exists (unexpected)\n")
	}

	// Check if deleted version exists
	_, err = s3Client.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(deletedKey),
	})
	if err != nil {
		fmt.Printf("✗ Deleted file does not exist: %v\n", err)
	} else {
		fmt.Printf("✓ Deleted file exists with .deleted suffix\n")
	}

	// Test 12: Cleanup - permanently delete the .deleted file
	fmt.Println("\n--- Test 12: Cleanup ---")
	_, err = s3Client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(deletedKey),
	})
	if err != nil {
		fmt.Printf("Error cleaning up deleted file: %v\n", err)
	} else {
		fmt.Printf("✓ Cleanup complete - deleted file removed\n")
	}

	fmt.Println("\n--- All Tests Complete ---")
	fmt.Println("All S3 operations have been tested using AWS SDK v1 (no chunked encoding).")
}
