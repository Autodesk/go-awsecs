# go-awsecs

Library and tools for AWS ECS operations.

# tools

## update-aws-ecs-service

Get:

```
go get -u github.com/andresvia/go-awsecs/cmd/update-aws-ecs-service
```

Use:

```
update-aws-ecs-service -h
Usage of update-aws-ecs-service:
  -cluster string
    	cluster
  -desired-count int
    	desired-count (default -1)
  -name-env value
    	name-env
  -name-image value
    	name-image
  -service string
    	service
```

Example:

```
AWS_PROFILE=myprofile AWS_REGION=myregion update-aws-ecs-service -cluster mycluster -service myservice -name-image mycontainer=newrepo/newimg:newtag -name-env mycontainer=envvarname=envvarvalue -desired-count 1
```
