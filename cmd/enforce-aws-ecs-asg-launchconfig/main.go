package main

import (
	"flag"
	"github.com/Autodesk/go-awsecs"
	"github.com/aws/aws-sdk-go/aws"
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
	profile := flag.String("profile", "", "profile name")
	region := flag.String("region", "", "region name")
	flag.Parse()

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Profile: *profile,
	}))

	if *region != "" {
		sess = sess.Copy(&aws.Config{Region: region})
	}

	elc := awsecs.EnforceLaunchConfig{
		ECSAPI:         *ecs.New(sess),
		ASAPI:          *autoscaling.New(sess),
		EC2API:         *ec2.New(sess),
		ASGName:        *asg,
		ECSClusterName: *cluster,
		BackOff:        backoff.NewExponentialBackOff(),
	}

	if err := elc.Apply(); err != nil {
		log.Fatal(err)
	}
}
