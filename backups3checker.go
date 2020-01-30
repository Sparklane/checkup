package checkup

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// BackupS3Checker implements a Checker for S3.
type BackupS3Checker struct {
	// Name is the name of the endpoint.
	Name string `json:"endpoint_name"`

	// Region of the bucket.
	// Default is eu-west-1
	Region string `json:"region"`

	// Name of the bucket.
	BucketName string `json:"bucket_name"`

	// Prefix in the bucket. Default is empty string.
	BucketPrefix string `json:"bucket_prefix,omitempty"`

	// MinAgeThreshold
	// Default is 36 hours.
	MinAgeThreshold string `json:"min_age_threshold,omitempty"`

	// MinSizeThreshold
	// Default is 1Mo
	MinSizeThreshold int64 `json:"min_size_threshold,omitempty"`

	s3Service *s3.S3
}

// Check performs checks using c according to its configuration.
// An error is only returned if there is a configuration error.
func (c BackupS3Checker) Check() (Result, error) {
	if c.Region == "" {
		c.Region = "eu-west-1"
	}
	var oldThreshold time.Duration
	if c.MinAgeThreshold == "" {
		c.MinAgeThreshold = "36h"
	}
	oldThreshold, _ = time.ParseDuration(c.MinAgeThreshold)
	if c.MinSizeThreshold == 0 {
		c.MinSizeThreshold = 1 * 1024 * 1024 // 1Mo
	}

	c.s3Service = s3.New(session.New(&aws.Config{Region: aws.String(c.Region)}))

	result := Result{Title: c.Name, Endpoint: c.Name, Timestamp: Timestamp()}
	result.Times = make(Attempts, 1)

	// https://github.com/awsdocs/aws-doc-sdk-examples/blob/master/go/example_code/s3/s3_list_objects.go
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(c.BucketName),
		Prefix: aws.String(c.BucketPrefix),
	}
	resp, err := c.s3Service.ListObjectsV2(input)
	if err != nil {
		return result, err
	}
	var lastItem *s3.Object = nil
	for _, item := range resp.Contents {
		if lastItem == nil || (*lastItem.LastModified).Before(*item.LastModified) {
			lastItem = item
		}
	}
	if lastItem == nil {
		result.Times[0].Error = "no backup"
	} else {
		if (*lastItem.LastModified).Before(time.Now().Add(-1 * oldThreshold)) {
			result.Times[0].Error = "no recent backup"
		}
		if *lastItem.Size < c.MinSizeThreshold {
			result.Times[0].Error = "size too low"
		}
	}

	return c.conclude(result), nil
}

// conclude takes the data in result from the attempts and
// computes remaining values needed to fill out the result.
func (c BackupS3Checker) conclude(result Result) Result {
	// Check errors (down)
	for i := range result.Times {
		if result.Times[i].Error != "" {
			result.Down = true
			return result
		}
	}

	result.Healthy = true
	return result
}
