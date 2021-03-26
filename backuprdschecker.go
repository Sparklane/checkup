package checkup

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
)

// BackupRDSChecker implements a Checker for ami.
type BackupRDSChecker struct {
	// Name is the name of the endpoint.
	Name string `json:"endpoint_name"`

	// Region of the bucket.
	// Default is eu-west-1
	Region string `json:"region"`

	// rds instance.
	Instance string `json:"instance"`

	// MinAgeThreshold
	// Default is 36 hours.
	MinAgeThreshold string `json:"min_age_threshold,omitempty"`

	rdsService *rds.RDS
}

// Check performs checks using c according to its configuration.
// An error is only returned if there is a configuration error.
func (c BackupRDSChecker) Check() (Result, error) {
	if c.Region == "" {
		c.Region = "eu-west-1"
	}
	var oldThreshold time.Duration
	if c.MinAgeThreshold == "" {
		c.MinAgeThreshold = "36h"
	}
	oldThreshold, _ = time.ParseDuration(c.MinAgeThreshold)

	c.rdsService = rds.New(session.New(&aws.Config{Region: aws.String(c.Region)}))

	result := Result{Title: c.Name, Endpoint: c.Name, Timestamp: Timestamp()}
	result.Times = make(Attempts, 1)

	// https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#RDS.DescribeDBSnapshots
	input := &rds.DescribeDBSnapshotsInput{
		DBInstanceIdentifier: aws.String(c.Instance),
		IncludePublic:        aws.Bool(false),
		IncludeShared:        aws.Bool(false),
	}
	resp, err := c.rdsService.DescribeDBSnapshots(input)
	if err != nil {
		return result, err
	}
	var lastSnapshot *rds.DBSnapshot = nil
	for _, snapshot := range resp.DBSnapshots {
		if *snapshot.Status == "available" {
			if lastSnapshot == nil || (*lastSnapshot.SnapshotCreateTime).Before(*snapshot.SnapshotCreateTime) {
				lastSnapshot = snapshot
			}
		}
	}
	if lastSnapshot == nil {
		result.Times[0].Error = "no backup"
	} else {
		if (*lastSnapshot.SnapshotCreateTime).Before(time.Now().Add(-1 * oldThreshold)) {
			result.Times[0].Error = "no recent backup"
		}
	}

	return c.conclude(result), nil
}

// conclude takes the data in result from the attempts and
// computes remaining values needed to fill out the result.
func (c BackupRDSChecker) conclude(result Result) Result {
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
