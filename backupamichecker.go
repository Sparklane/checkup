package checkup

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// BackupAMIChecker implements a Checker for ami.
type BackupAMIChecker struct {
	// Name is the name of the endpoint.
	Name string `json:"endpoint_name"`

	// Region of the bucket.
	// Default is eu-west-1
	Region string `json:"region"`

	// Prefix of the ami.
	AmiPrefix string `json:"ami_prefix"`

	// MinAgeThreshold
	// Default is 36 hours.
	MinAgeThreshold string `json:"min_age_threshold,omitempty"`

	ec2Service *ec2.EC2
}

// Check performs checks using c according to its configuration.
// An error is only returned if there is a configuration error.
func (c BackupAMIChecker) Check() (Result, error) {
	if c.Region == "" {
		c.Region = "eu-west-1"
	}
	var oldThreshold time.Duration
	if c.MinAgeThreshold == "" {
		c.MinAgeThreshold = "36h"
	}
	oldThreshold, _ = time.ParseDuration(c.MinAgeThreshold)

	c.ec2Service = ec2.New(session.New(&aws.Config{Region: aws.String(c.Region)}))

	result := Result{Title: c.Name, Endpoint: c.Name, Timestamp: Timestamp()}
	result.Times = make(Attempts, 1)

	// https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#EC2.DescribeImages
	input := &ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("name"),
				Values: []*string{aws.String(c.AmiPrefix + "*")},
			},
		},
	}
	resp, err := c.ec2Service.DescribeImages(input)
	if err != nil {
		return result, err
	}
	var lastAmi *ec2.Image = nil
	var lastAmiCreationDate time.Time
	for _, image := range resp.Images {
		if *image.State == "available" {
			creationDate, _ := time.Parse("2006-01-02T15:04:05.000Z", *image.CreationDate)
			if lastAmi == nil || lastAmiCreationDate.Before(creationDate) {
				lastAmi = image
				lastAmiCreationDate = creationDate
			}
		}
	}
	if lastAmi == nil {
		result.Times[0].Error = "no backup"
	} else {
		if lastAmiCreationDate.Before(time.Now().Add(-1 * oldThreshold)) {
			result.Times[0].Error = "no recent backup"
		}
	}

	return c.conclude(result), nil
}

// conclude takes the data in result from the attempts and
// computes remaining values needed to fill out the result.
func (c BackupAMIChecker) conclude(result Result) Result {
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
