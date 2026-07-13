package saws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
)

var _AWS *AWS

// AWS represents all the services used by Stormkit.
type AWS struct {
	Config  *aws.Config
	Session *session.Session
	Lambda  lambdaiface.LambdaAPI
	STS     stsiface.STSAPI
	S3      s3iface.S3API
	EC2     ec2iface.EC2API
	RDS     rdsiface.RDSAPI
	Route53 route53iface.Route53API
}

func createSession(region string, awsCfg *aws.Config) (*session.Session, error) {
	var cfg *aws.Config

	if config.IsDevelopment() || config.IsTest() {
		cfg = &aws.Config{
			Region:           aws.String(region),
			Credentials:      credentials.NewStaticCredentials("static", "empty-credentials", ""),
			S3ForcePathStyle: aws.Bool(true),
		}
	} else {
		cfg = awsCfg
	}

	return session.NewSession(cfg)
}

// configure AWS.
func configure(region string, cfgs ...*aws.Config) *AWS {
	var err error

	a := &AWS{}

	if len(cfgs) > 0 {
		a.Config = cfgs[0]
	} else {
		a.Config = &aws.Config{Region: aws.String(region)}
	}

	a.Session, err = createSession(region, a.Config)

	if err != nil {
		panic(err)
	}

	a.Lambda = newLambda(a.Session, cfgs...)
	a.RDS = newRDS(a.Session, cfgs...)
	a.Route53 = newRoute53(a.Session)

	return a
}

// Instance returns the aws instance.
func Instance() *AWS {
	if _AWS == nil {
		_AWS = configure(config.Get().AWS.Region)
	}

	return _AWS
}

// SetInstance sets the AWS instance to a custom one. It is useful for unit-testing.
func SetInstance(a *AWS) {
	_AWS = a
}
