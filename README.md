# go-awsecs

Library and tools for AWS ECS operations.

# tools

## update-aws-ecs-service

This tool is inspired by [AWS CodePipeline image definitions file method for updating existing ECS services](https://docs.aws.amazon.com/codepipeline/latest/userguide/pipelines-create.html#pipelines-create-image-definitions), this tool attempts to do something similar, in a standalone fashion without depending on AWS CodePipeline.

Get:

```
go get -u github.com/andresvia/go-awsecs/cmd/update-aws-ecs-service
```

Use:

```
update-aws-ecs-service -h
Usage of update-aws-ecs-service:
  -cluster string
    	cluster name
  -container-envvar value
    	container-name=envvar-name=envvar-value
  -container-image value
    	container-name=image
  -desired-count int
    	desired-count (negative: no change) (default -1)
  -service string
    	service name
```

Example, first, build and push a new Docker image for your service somewhere else.

```
docker build -t myrepo/myimg:newtag .
docker push myrepo/myimg:newtag
```

Then, alter the existing container image only, like AWS CodePipeline does.

```
AWS_PROFILE=myprofile AWS_REGION=myregion update-aws-ecs-service \
  -cluster mycluster \
  -service myservice \
  -container-image mycontainer=myrepo/myimg:newtag
```

Alternatively, you can also alter, environment variables and service desired count.

```
AWS_PROFILE=myprofile AWS_REGION=myregion update-aws-ecs-service \
  -cluster mycluster \
  -service myservice \
  -container-image mycontainer=myrepo/myimg:newtag \
  -container-envvar mycontainer=envvarname=envvarvalue \
  -desired-count 1
```

## enforce-aws-ecs-asg-launchconfig

This tool is useful to ensure that all EC2 instances in a ECS cluster backed up by a ASG, share the launch configuration defined in the ASG. This tool doesn't work with launch templates. ECS EC2 Container Instances will be drained. EC2 Instances will be terminated (after they are drained).

Get:

```
go get -u github.com/andresvia/go-awsecs/cmd/enforce-aws-ecs-asg-launchconfig
```

Use:

```
enforce-aws-ecs-asg-launchconfig -h
Usage of enforce-aws-ecs-asg-launchconfig:
  -asg string
    	asg name
  -cluster string
    	cluster name
```
