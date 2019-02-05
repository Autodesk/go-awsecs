#!/bin/sh

set -e

cd cmd/update-aws-ecs-service
go get -f -u -v
cd ../../

cd cmd/enforce-aws-ecs-asg-launchconfig
go get -f -u -v
cd ../../

cd cmd/update-aws-ecs-service
GOARCH=amd64 GOOS=windows go build -o update-aws-ecs-service.exe
zip update-aws-ecs-service-amd64-windows.zip update-aws-ecs-service.exe
rm update-aws-ecs-service.exe
mv -f update-aws-ecs-service-amd64-windows.zip ../../
cd ../../

cd cmd/update-aws-ecs-service
GOARCH=amd64 GOOS=darwin go build -o update-aws-ecs-service
zip update-aws-ecs-service-amd64-darwin.zip update-aws-ecs-service
rm update-aws-ecs-service
mv -f update-aws-ecs-service-amd64-darwin.zip ../../
cd ../../

cd cmd/update-aws-ecs-service
GOARCH=amd64 GOOS=linux go build -o update-aws-ecs-service
zip update-aws-ecs-service-amd64-linux.zip update-aws-ecs-service
rm update-aws-ecs-service
mv -f update-aws-ecs-service-amd64-linux.zip ../../
cd ../../

cd cmd/enforce-aws-ecs-asg-launchconfig
GOARCH=amd64 GOOS=windows go build -o enforce-aws-ecs-asg-launchconfig.exe
zip enforce-aws-ecs-asg-launchconfig-amd64-windows.zip enforce-aws-ecs-asg-launchconfig.exe
rm enforce-aws-ecs-asg-launchconfig.exe
mv -f enforce-aws-ecs-asg-launchconfig-amd64-windows.zip ../../
cd ../../

cd cmd/enforce-aws-ecs-asg-launchconfig
GOARCH=amd64 GOOS=darwin go build -o enforce-aws-ecs-asg-launchconfig
zip enforce-aws-ecs-asg-launchconfig-amd64-darwin.zip enforce-aws-ecs-asg-launchconfig
rm enforce-aws-ecs-asg-launchconfig
mv -f enforce-aws-ecs-asg-launchconfig-amd64-darwin.zip ../../
cd ../../

cd cmd/enforce-aws-ecs-asg-launchconfig
GOARCH=amd64 GOOS=linux go build -o enforce-aws-ecs-asg-launchconfig
zip enforce-aws-ecs-asg-launchconfig-amd64-linux.zip enforce-aws-ecs-asg-launchconfig
rm enforce-aws-ecs-asg-launchconfig
mv -f enforce-aws-ecs-asg-launchconfig-amd64-linux.zip ../../
cd ../../
