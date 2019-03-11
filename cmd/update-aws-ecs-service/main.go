package main

import (
	"flag"
	"fmt"
	"git.autodesk.com/t-villa/go-awsecs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
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
	desiredCount := flag.Int64("desired-count", -1, "desired-count (negative: no change)")

	var images mapFlag = map[string]string{}
	var envs mapMapFlag = map[string]map[string]string{}
	var secrets mapMapFlag = map[string]map[string]string{}

	flag.Var(&images, "container-image", "container-name=image")
	flag.Var(&envs, "container-envvar", "container-name=envvar-name=envvar-value")
	flag.Var(&secrets, "container-secret", "container-name=secret-name=secret-valuefrom")
	flag.Parse()

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Profile: *profile,
	}))

	if *region != "" {
		sess = sess.Copy(&aws.Config{Region: region})
	}

	esu := awsecs.ECSServiceUpdate{
		API:          *ecs.New(sess),
		Cluster:      *cluster,
		Service:      *service,
		Image:        images,
		Environment:  envs,
		Secrets:      secrets,
		DesiredCount: int64ptr(*desiredCount),
		BackOff:      backoff.NewExponentialBackOff(),
	}

	if err := esu.Apply(); err != nil {
		if err != awsecs.ErrFailedRollback {
			log.Fatal(err)
		} else {
			os.Exit(1)
		}
	}
}
