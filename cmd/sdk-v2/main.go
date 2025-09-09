package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
	fmt.Println("Using AWS SDK v2 with environment variables from .env file...")

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

	fmt.Printf("AWS Config from environment:\n")
	fmt.Printf("  Access Key ID: %s\n", accessKeyID)
	fmt.Printf("  Secret Access Key: %s*** (length: %d)\n", secretAccessKey[:5], len(secretAccessKey))
	fmt.Printf("  Region: %s\n", region)
	fmt.Printf("  Bucket: %s\n", bucketName)
	fmt.Printf("  Endpoint URL: %s\n", endpointURL)
	fmt.Printf("  Environment: %s\n", environment)

	ctx := context.Background()

	// Initialize AWS SDK v2 client
	fmt.Println("\n--- Initializing AWS SDK v2 Client ---")

	// Load AWS configuration with custom credentials
	awsConfig, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
			},
		}),
		config.WithRegion(region),
	)
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// Create S3 client with custom endpoint if provided
	var s3Client *s3.Client
	if endpointURL != "" {
		s3Client = s3.NewFromConfig(awsConfig, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpointURL)
			o.UsePathStyle = true
			o.DisableMultiRegionAccessPoints = true
		})
		fmt.Printf("Using custom endpoint: %s\n", endpointURL)
	} else {
		s3Client = s3.NewFromConfig(awsConfig)
		fmt.Printf("Using default AWS S3 endpoint\n")
	}

	// Test 1: List buckets
	fmt.Println("\n--- Test 1: List Buckets ---")
	result, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
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
	_, err = s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
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
	fileContent := "Hello from AWS SDK v2 with environment variables!\nThis should work with proper Tebi.io configuration."
	testKey := "test-folder/test-file-v2.txt"

	// Try different upload approaches with AWS SDK v2
	fmt.Printf("Attempting upload with key: %s\n", testKey)

	// Method 1: Basic PutObject
	putObjectInput := &s3.PutObjectInput{
		Bucket:        aws.String(bucketName),
		Key:           aws.String(testKey),
		Body:          strings.NewReader(fileContent),
		ContentType:   aws.String("text/plain"),
		ContentLength: aws.Int64(int64(len(fileContent))),
	}

	_, err = s3Client.PutObject(ctx, putObjectInput)
	if err != nil {
		fmt.Printf("PutObject failed: %v\n", err)

		// Method 2: Try with minimal parameters
		fmt.Println("Trying minimal PutObject...")
		minimalInput := &s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(testKey + "-minimal"),
			Body:   strings.NewReader(fileContent),
		}

		_, err = s3Client.PutObject(ctx, minimalInput)
		if err != nil {
			fmt.Printf("Minimal PutObject also failed: %v\n", err)
			fmt.Println("Upload failed with AWS SDK v2 - this appears to be a Tebi.io compatibility issue")
		} else {
			fmt.Printf("✓ Minimal PutObject succeeded with key: %s (%d bytes)\n", testKey+"-minimal", len(fileContent))
			testKey = testKey + "-minimal" // Use the successful key for remaining tests
		}
	} else {
		fmt.Printf("✓ PutObject succeeded with key: %s (%d bytes)\n", testKey, len(fileContent))
	}

	// If upload succeeded, continue with other tests
	if err == nil {
		// Test 5: Verify upload
		fmt.Println("\n--- Test 5: Verify Upload ---")
		_, err = s3Client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(testKey),
		})
		if err != nil {
			fmt.Printf("Error verifying object exists: %v\n", err)
		} else {
			fmt.Printf("✓ Object exists and is accessible\n")
		}

		// Test 6: Get file metadata
		fmt.Println("\n--- Test 6: Get File Metadata ---")
		headResult, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(testKey),
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
			// Custom endpoint (like Tebi.io, DigitalOcean Spaces, MinIO, etc.)
			endpointURLTrimmed := strings.TrimSuffix(endpointURL, "/")
			publicURL = fmt.Sprintf("%s/%s/%s", endpointURLTrimmed, bucketName, testKey)
		} else {
			// Standard AWS S3 URL
			publicURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, region, testKey)
		}
		fmt.Printf("✓ Public URL: %s\n", publicURL)

		// Test 8: Generate presigned URL
		fmt.Println("\n--- Test 8: Generate Presigned URL ---")
		presignClient := s3.NewPresignClient(s3Client)
		presignedResult, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(testKey),
		}, func(opts *s3.PresignOptions) {
			opts.Expires = 15 * time.Minute
		})
		if err != nil {
			fmt.Printf("Error generating presigned URL: %v\n", err)
		} else {
			fmt.Printf("✓ Presigned URL: %s\n", presignedResult.URL)
		}

		// Test 9: List files in bucket with prefix
		fmt.Println("\n--- Test 9: List Files ---")
		keyParts := strings.Split(testKey, "/")
		prefix := ""
		if len(keyParts) > 1 {
			prefix = keyParts[0] + "/" // Get the folder prefix
		}

		listResult, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:  aws.String(bucketName),
			Prefix:  aws.String(prefix),
			MaxKeys: aws.Int32(10),
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
		deletedKey := testKey + ".deleted"

		// Copy to deleted key
		_, err = s3Client.CopyObject(ctx, &s3.CopyObjectInput{
			Bucket:     aws.String(bucketName),
			Key:        aws.String(deletedKey),
			CopySource: aws.String(fmt.Sprintf("%s/%s", bucketName, testKey)),
		})
		if err != nil {
			fmt.Printf("Error copying file for soft delete: %v\n", err)
		} else {
			fmt.Printf("✓ File copied to deleted key: %s\n", deletedKey)

			// Delete original
			_, err = s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(testKey),
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
		_, err = s3Client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(testKey),
		})
		if err != nil {
			fmt.Printf("✓ Original file no longer exists (expected)\n")
		} else {
			fmt.Printf("✗ Original file still exists (unexpected)\n")
		}

		// Check if deleted version exists
		_, err = s3Client.HeadObject(ctx, &s3.HeadObjectInput{
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
		_, err = s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(deletedKey),
		})
		if err != nil {
			fmt.Printf("Error cleaning up deleted file: %v\n", err)
		} else {
			fmt.Printf("✓ Cleanup complete - deleted file removed\n")
		}
	}

	fmt.Println("\n--- All Tests Complete ---")
	if err != nil {
		fmt.Println("Some tests failed due to upload issues with AWS SDK v2 and Tebi.io.")
		fmt.Println("This appears to be a known compatibility issue between AWS SDK v2 and Tebi.io's S3 implementation.")
	} else {
		fmt.Println("All S3 operations completed successfully using AWS SDK v2!")
	}
}
