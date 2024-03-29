package main

import (
	"flag"
	"fmt"
	"github.com/Autodesk/go-awsecs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/cenkalti/backoff"
	"log"
	"os"
	"strings"
)

func int64ptr(x int64) *int64 {
	if x < 0 {
		return nil
	}
	return &x
}

func keyEqValue(kv string) (string, string) {
	parts := strings.SplitN(kv, "=", 2)
	return strings.TrimSpace(strings.Join(parts[0:1], "")), strings.TrimSpace(strings.Join(parts[1:2], ""))
}

type mapFlag map[string]string

type mapMapFlag map[string]map[string]string

func (kvs *mapFlag) String() string {
	return fmt.Sprintf("%v", *kvs)
}

func (kvs *mapMapFlag) String() string {
	return fmt.Sprintf("%v", *kvs)
}

func (kvs mapFlag) Set(value string) error {
	key, value := keyEqValue(value)
	kvs[key] = value
	return nil
}

func (kvs mapMapFlag) Set(value string) error {
	key, value := keyEqValue(value)
	valueKey, value := keyEqValue(value)
	if kvs[key] == nil {
		kvs[key] = map[string]string{}
	}
	kvs[key][valueKey] = value
	return nil
}

func main() {
	cluster := flag.String("cluster", "", "cluster name")
	service := flag.String("service", "", "service name")
	profile := flag.String("profile", "", "profile name")
	region := flag.String("region", "", "region name")
	taskdef := flag.String("taskdef", "", "base task definition (instead of current)")
	desiredCount := flag.Int64("desired-count", -1, "desired-count (negative: no change)")
	taskrole := flag.String("task-role", "", fmt.Sprintf(`task iam role, set to "%s" to clear`, awsecs.TaskRoleKnockoutValue))
	waituntil := flag.String("wait-until", awsecs.WaitUntilPrimaryRolled, fmt.Sprintf("valid options are: %s", strings.Join(awsecs.WaitUntilOptionList, ", ")))

	var images mapFlag = map[string]string{}
	var envs mapMapFlag = map[string]map[string]string{}
	var secrets mapMapFlag = map[string]map[string]string{}
	var logopts mapMapMapFlag = map[string]map[string]map[string]string{}
	var logsecrets mapMapMapFlag = map[string]map[string]map[string]string{}

	flag.Var(&images, "container-image", "container-name=image")
	flag.Var(&envs, "container-envvar", "container-name=envvar-name=envvar-value")
	flag.Var(&secrets, "container-secret", "container-name=secret-name=secret-valuefrom")
	flag.Var(&logopts, "container-logopt", "container-name=logdriver=logopt=value")
	flag.Var(&logsecrets, "container-logsecret", "container-name=logdriver=logsecret=valuefrom")
	flag.Parse()

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Profile: *profile,
	}))

	if *region != "" {
		sess = sess.Copy(&aws.Config{Region: region})
	}

	esu := awsecs.ECSServiceUpdate{
		EcsApi:           ecs.New(sess),
		ElbApi:           elbv2.New(sess),
		Cluster:          *cluster,
		Service:          *service,
		Image:            images,
		Environment:      envs,
		Secrets:          secrets,
		LogDriverOptions: logopts,
		LogDriverSecrets: logsecrets,
		TaskRole:         *taskrole,
		DesiredCount:     int64ptr(*desiredCount),
		Taskdef:          *taskdef,
		WaitUntil:        waituntil,
		BackOff:          backoff.NewExponentialBackOff(),
	}

	if err := esu.Apply(); err != nil {
		if err != awsecs.ErrFailedRollback {
			log.Fatal(err)
		} else {
			os.Exit(1)
		}
	}
}
