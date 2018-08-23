package main

import (
	"flag"
	"github.com/andresvia/go-awsecs"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/cenkalti/backoff"
	"log"
)

func main() {
	cluster := flag.String("cluster", "", "cluster name")
	asg := flag.String("asg", "", "asg name")
	flag.Parse()

	session := session.Must(session.NewSession())

	elc := awsecs.EnforceLaunchConfig{
		ECSAPI:         *ecs.New(session),
		ASAPI:          *autoscaling.New(session),
		EC2API:         *ec2.New(session),
		ASGName:        *asg,
		ECSClusterName: *cluster,
		BackOff:        backoff.NewExponentialBackOff(),
	}

	if err := elc.Apply(); err != nil {
		log.Fatal(err)
	}
}
