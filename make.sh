#!/bin/sh

set -e

cd cmd/update-aws-ecs-service
go get -v
GOARCH=amd64 GOOS=darwin go build -o update-aws-ecs-service-amd64-darwin
zip update-aws-ecs-service-amd64-darwin.zip update-aws-ecs-service-amd64-darwin
rm update-aws-ecs-service-amd64-darwin
mv update-aws-ecs-service-amd64-darwin.zip ../../
cd ../../

cd cmd/update-aws-ecs-service
go get -v
GOARCH=amd64 GOOS=linux go build -o update-aws-ecs-service-amd64-linux
zip update-aws-ecs-service-amd64-linux.zip update-aws-ecs-service-amd64-linux
rm update-aws-ecs-service-amd64-linux
cp update-aws-ecs-service-amd64-linux.zip ../../
cd ../../

cd cmd/enforce-aws-ecs-asg-launchconfig
go get -v
GOARCH=amd64 GOOS=darwin go build -o enforce-aws-ecs-asg-launchconfig-amd64-darwin
zip enforce-aws-ecs-asg-launchconfig-amd64-darwin.zip enforce-aws-ecs-asg-launchconfig-amd64-darwin
rm enforce-aws-ecs-asg-launchconfig-amd64-darwin
mv enforce-aws-ecs-asg-launchconfig-amd64-darwin.zip ../../
cd ../../

cd cmd/enforce-aws-ecs-asg-launchconfig
go get -v
GOARCH=amd64 GOOS=linux go build -o enforce-aws-ecs-asg-launchconfig-amd64-linux
zip enforce-aws-ecs-asg-launchconfig-amd64-linux.zip enforce-aws-ecs-asg-launchconfig-amd64-linux
rm enforce-aws-ecs-asg-launchconfig-amd64-linux
cp enforce-aws-ecs-asg-launchconfig-amd64-linux.zip ../../
cd ../../
