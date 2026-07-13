package saws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
)

// newRoute53 returns a new Route53 instance.
func newRoute53(sess *session.Session) route53iface.Route53API {
	return route53iface.Route53API(route53.New(sess))
}
