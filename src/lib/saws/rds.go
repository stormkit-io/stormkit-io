package saws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
)

// APIDBIdentifier is the identifier for the API database.
const APIDBIdentifier = "terraform-20181125143740773000000003"

// newRDS returns a new RDS instance.
func newRDS(sess *session.Session, cfgs ...*aws.Config) rdsiface.RDSAPI {
	return rdsiface.RDSAPI(rds.New(sess, cfgs...))
}

// CreateSnapshot creates a snapshot of the database.
func (a *AWS) CreateSnapshot(dbIdenfitier string) (*rds.CreateDBSnapshotOutput, error) {
	input := &rds.CreateDBSnapshotInput{
		DBInstanceIdentifier: aws.String(dbIdenfitier),
		DBSnapshotIdentifier: aws.String(time.Now().Format("sn-20060102150405")),
	}

	return a.RDS.CreateDBSnapshot(input)
}

// DeleteSnapshot creates a snapshot of the database.
func (a *AWS) DeleteSnapshot(identifier string) (*rds.DeleteDBSnapshotOutput, error) {
	input := &rds.DeleteDBSnapshotInput{
		DBSnapshotIdentifier: aws.String(identifier),
	}

	return a.RDS.DeleteDBSnapshot(input)
}
