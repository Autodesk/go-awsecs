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
AWS_PROFILE=myprofile AWS_REGION=myregion update-aws-ecs-service -cluster mycluster -service myservice -container-image mycontainer=myrepo/myimg:newtag
```

Alternatively, you can also alter, environment variables and service desired count.

```
AWS_PROFILE=myprofile AWS_REGION=myregion update-aws-ecs-service -cluster mycluster -service myservice -container-image mycontainer=myrepo/myimg:newtag -container-envvar mycontainer=envvarname=envvarvalue -desired-count 1
```
