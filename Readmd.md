# Tebi.io AWS SDK Go Compatibility Test

This repository demonstrates a compatibility issue between Tebi.io's S3-compatible API and AWS SDK for Go v2. The same credentials and configuration work perfectly with AWS SDK v1 but fail with AWS SDK v2.

## Issue Description

When using AWS SDK for Go v2 with Tebi.io, file uploads fail while the same credentials and configuration work perfectly with AWS SDK for Go v1. This appears to be related to how AWS SDK v2 handles HTTP requests differently (particularly around chunked encoding and request signing).

**Working**: AWS SDK Go v1 ✅  
**Not Working**: AWS SDK Go v2 ❌

## Repository Structure

```
├── cmd/
│   ├── sdk-v1/           # Working example using AWS SDK v1
│   │   └── main.go
│   └── sdk-v2/           # Failing example using AWS SDK v2
│       └── main.go
├── .env.example          # Environment variables template
├── go.mod               # Go module with both SDK versions
└── README.md            # This file
```

## Quick Start

### Prerequisites

- Go 1.21 or later
- Valid Tebi.io credentials
- Access to a Tebi.io bucket

### Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/imzza/tebi-aws-sdk-go-examples.git
   cd tebi-aws-sdk-go-examples
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Configure environment variables**
   ```bash
   cp .env.example .env
   ```
   
   Edit `.env` with your Tebi.io credentials:
   ```bash
   ENVIRONMENT=dev
   AWS_ACCESS_KEY_ID=your_tebi_access_key
   AWS_SECRET_ACCESS_KEY=your_tebi_secret_key
   AWS_DEFAULT_REGION=your_tebi_region
   AWS_BUCKET_NAME=your_bucket_name
   AWS_ENDPOINT_URL=https://your-endpoint.tebi.io
   ```

### Running the Examples

#### Test with AWS SDK v1 (Working)
```bash
go run cmd/sdk-v1/main.go
```

**Expected Result**: All tests should pass ✅
- Lists buckets successfully
- Uploads files without issues
- Performs all S3 operations correctly

#### Test with AWS SDK v2 (Not Working)
```bash
go run cmd/sdk-v2/main.go
```

**Expected Result**: Upload operations will fail ❌
- Bucket listing may work
- File upload operations fail
- Error messages related to request signing or HTTP protocol

## Test Operations

Both examples perform identical operations to demonstrate the compatibility difference:

1. **List Buckets** - Verify connection and credentials
2. **Head Bucket** - Check bucket existence and permissions
3. **Generate File Key** - Create unique file paths
4. **Upload File** - Core operation that fails in v2
5. **Verify Upload** - Confirm file was uploaded
6. **Get Metadata** - Retrieve file information
7. **Generate URLs** - Public and presigned URL generation
8. **List Files** - Browse bucket contents
9. **Soft Delete** - Copy and delete operations
10. **Cleanup** - Remove test files

## Key Differences

### AWS SDK v1 Configuration
```go
sess, err := session.NewSession(&aws.Config{
    Region: aws.String(region),
    Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
    Endpoint: aws.String(endpointURL),
    S3ForcePathStyle: aws.Bool(true),
})
s3Client := s3.New(sess)
```

### AWS SDK v2 Configuration
```go
awsConfig, err := config.LoadDefaultConfig(ctx,
    config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
        Value: aws.Credentials{
            AccessKeyID:     accessKeyID,
            SecretAccessKey: secretAccessKey,
        },
    }),
    config.WithRegion(region),
)

s3Client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
    o.BaseEndpoint = aws.String(endpointURL)
    o.UsePathStyle = true
    o.DisableMultiRegionAccessPoints = true
})
```

## Troubleshooting

### Common Issues

1. **Environment Variables Not Found**
   - Ensure `.env` file is in the root directory
   - Check that all required variables are set
   - Verify `.env` file is not ignored by git

2. **Connection Refused**
   - Verify `AWS_ENDPOINT_URL` is correct
   - Check firewall and network settings
   - Ensure Tebi.io endpoint is accessible

3. **Authentication Errors**
   - Double-check access key and secret key
   - Verify credentials have proper permissions
   - Check region configuration

### Debug Mode

Both examples include detailed logging. For additional debugging, you can:

1. **Enable AWS SDK logging** (already enabled in examples)
2. **Check network traffic** with tools like Wireshark
3. **Compare HTTP requests** between v1 and v2

## Expected Behavior

### SDK v1 Output (Working)
```
✓ Successfully listed buckets: 2 buckets found
✓ Bucket 'your-bucket' exists and is accessible
✓ File uploaded successfully with key: test-folder/test-file.txt (58 bytes)
✓ Object exists and is accessible
✓ All operations completed successfully
```

### SDK v2 Output (Not Working)
```
✓ Successfully listed buckets: 2 buckets found
✓ Bucket 'your-bucket' exists and is accessible
✗ PutObject failed: operation error S3: PutObject, ...
✗ Upload failed with AWS SDK v2 - this appears to be a Tebi.io compatibility issue
```

## For Tebi.io Support Team

This repository demonstrates the exact issue we're experiencing. To reproduce:

1. Use your own Tebi.io test credentials
2. Run both examples with identical configuration
3. Observe that v1 works while v2 fails

The issue appears to be related to:
- HTTP request signing differences between SDK versions
- Chunked transfer encoding handling
- Request header variations

## Dependencies

- `github.com/aws/aws-sdk-go v1.55.8` - AWS SDK v1
- `github.com/aws/aws-sdk-go-v2 v1.39.0` - AWS SDK v2 (and related packages)
- `github.com/joho/godotenv v1.5.1` - Environment variable loading
- `github.com/matoous/go-nanoid/v2 v2.1.0` - Unique ID generation

## Security Notes

- Never commit `.env` files with real credentials
- Use environment-specific configuration
- Rotate credentials regularly
- Follow principle of least privilege for bucket permissions

## Support

If you're from the Tebi.io support team and need additional information:
- All sensitive data has been removed from this public repository
- Examples use environment variables for configuration
- Both examples are self-contained and ready to run
- Detailed logging is enabled to help with debugging

For questions or additional test cases, please let us know what specific scenarios you'd like us to test.