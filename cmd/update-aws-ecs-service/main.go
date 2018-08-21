package main

import (
	"flag"
	"fmt"
	"github.com/andresvia/go-awsecs"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/cenkalti/backoff"
	"log"
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

func (kvs *mapFlag) Set(value string) error {
	key, value := keyEqValue(value)
	kvs2 := *kvs
	if kvs2 == nil {
		kvs2 = map[string]string{}
	}
	kvs2[key] = value
	return nil
}

func (kvs *mapMapFlag) Set(value string) error {
	key, value := keyEqValue(value)
	valueKey, value := keyEqValue(value)
	kvs2 := *kvs
	if kvs2 == nil {
		kvs2 = map[string]map[string]string{}
	}
	kvs3 := kvs2[key]
	if kvs3 == nil {
		kvs3 = map[string]string{}
	}
	kvs3[valueKey] = value
	return nil
}

func main() {
	cluster := flag.String("cluster", "", "cluster name")
	service := flag.String("service", "", "service name")
	desiredCount := flag.Int64("desired-count", -1, "desired-count (negative: no change)")

	var images mapFlag
	var envs mapMapFlag

	flag.Var(&images, "container-image", "container-name=image")
	flag.Var(&envs, "container-envvar", "container-name=envvar-name=envvar-value")
	flag.Parse()

	esu := awsecs.ECSServiceUpdate{
		API:          *ecs.New(session.Must(session.NewSession())),
		Cluster:      *cluster,
		Service:      *service,
		Image:        images,
		Environment:  envs,
		DesiredCount: int64ptr(*desiredCount),
		BackOff:      backoff.NewExponentialBackOff(),
	}

	if err := esu.Apply(); err != nil {
		log.Fatal(err)
	}
}
