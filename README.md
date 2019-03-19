# go-awsecs

Library and tools for AWS ECS operations.

Get golang: https://golang.org/dl/

RTFM: https://github.com/golang/go/wiki/SettingGOPATH

# tools

## update-aws-ecs-service

Reliably update a single ECS service with a single simple discrete command.

![flowchart](update-aws-ecs-service.png)

Is a deployment tool inspired by [AWS CodePipeline image definitions file method for updating existing ECS services](https://docs.aws.amazon.com/codepipeline/latest/userguide/pipelines-create.html#pipelines-create-image-definitions). This tool is first and foremost an acknowledgment that orchestrating application deployments is a **hard problem** and does not attempt to solve that, instead, it tries to do something similar to AWS CodePipeline in a standalone fashion without depending on AWS CodePipeline, and more importantly without having to create individual AWS CodePipeline pipelines.

Get:

Grab binary distribution from [releases tab](https://git.autodesk.com/t-villa/go-awsecs/releases). Or.

```
go get -v -u git.autodesk.com/t-villa/go-awsecs/cmd/update-aws-ecs-service
```

Use<sup>1</sup>:

```
update-aws-ecs-service -h
Usage of update-aws-ecs-service:
  -cluster string
    	cluster name
  -container-envvar value
    	container-name=envvar-name=envvar-value
  -container-image value
    	container-name=image
  -container-secret value
    	container-name=secret-name=secret-valuefrom
  -desired-count int
    	desired-count (negative: no change) (default -1)
  -profile string
    	profile name
  -region string
    	region name
  -service string
    	service name
```

Example.

First, build and push a new Docker image for your service somewhere else.

```
docker build -t myrepo/myimg:newtag .
docker push myrepo/myimg:newtag
```

Then, alter the existing container image only, like AWS CodePipeline does.

```
update-aws-ecs-service \
  -cluster mycluster \
  -service myservice \
  -container-image mycontainer=myrepo/myimg:newtag
```

Alternatively, you can also alter environment variables and service desired count.

```
update-aws-ecs-service \
  -cluster mycluster \
  -service myservice \
  -container-image mycontainer=myrepo/myimg:newtag \
  -container-envvar mycontainer=envvarname=envvarvalue \
  -desired-count 1
```

💡 Use the empty value on `-container-envvar` or `-container-secret` to unset (K.O.) the environment variable or secret. Example.

```
update-aws-ecs-service \
  -cluster mycluster \
  -service myservice \
  -container-envvar mycontainer=myenvvarname= \
  -container-secret mycontainer=mysecretname= \
```

### update-aws-ecs-service compared to AWS CodePipeline

 - With `update-aws-ecs-service` there is no need to create individual AWS CodePipeline pipelines per service
 - `update-aws-ecs-service` allow updates of container definitions "Environment" and "[Secrets](https://aws.amazon.com/about-aws/whats-new/2018/11/aws-launches-secrets-support-for-amazon-elastic-container-servic/)"

### update-aws-ecs-service compared to AWS CLI

Although similar results can be achieved glueing multiple `awscli` commands, a single `update-aws-ecs-service` is different.

 - `aws ecs update-service` only invokes `UpdateService` which is an async call
 - `aws ecs wait services-stable` is not linked to the ECS Deployment Entity<sup>2</sup> returned by `UpdateService`
 - `update-aws-ecs-service` provides automatic rollback

### update-aws-ecs-service compared to Terraform

It is a [known issue](https://github.com/terraform-providers/terraform-provider-aws/issues/3107) that Terraform, does not wait for an ECS Service to be updated, a decision made probably by design by Hashicorp.

However, `update-aws-ecs-service` can be used in conjunction with Terraform, just keep in mind that when **provisioning** a service, start with an "initial task definition", and configure the lifecycle of the `task_definition` attribute to `ignore_changes`.

```
resource "aws_ecs_service" "my_service" {
  task_definition = "my_initial_task_def"
  // ...
  
  lifecycle {
    ignore_changes = ["task_definition" /* ... */]
  }
}
```

That way Terraform will be maintained as the "provisioning tool" and `update-aws-ecs-service` as the "deployment tool".

### update-aws-ecs-service compared to Terraform+scripts

[verified-terragrunt-apply](https://git.autodesk.com/t-villa/gb-sh-verified-terragrunt-apply) served as groundwork for `update-aws-ec-service`, it is now deprecated.

Other alternatives include:

 - Do `aws ecs wait services-stable` commands after the `terraform apply` command
 - Do `curl` commands after the `terraform apply` command until a desired result is obtained probably a number of times (works only for HTTP/HTTPS services with accesible endpoints) ([example](https://git.autodesk.com/EIS-EA-MOJO/deploy-lem-api-service/blob/16334841acf12a2796b033ae0f610ca2dd0ad311/Jenkinsfile#L678))

### update-aws-ecs-service compared to AWS CodeDeploy

TBC<sup>3</sup>.

### update-aws-ecs-service compared to amazon-ecs-cli

TBC.

### update-aws-ecs-service compared to ecs-deploy

The [ecs-deploy](https://github.com/silinternational/ecs-deploy) script [doesn't recognize multi-container tasks](https://github.com/silinternational/ecs-deploy/issues/132).

### update-aws-ecs-service compared to ecs-goploy

[ecs-goploy](https://github.com/h3poteto/ecs-goploy) as a re-implementation of ecs-deploy shares the same caveats.

### update-aws-ecs-service compared to Autodesk CloudOSv2

`update-aws-ecs-service` is just a tool to update existing AWS ECS services. You just need to know how to build Docker images.

More comparisons to be added.

## enforce-aws-ecs-asg-launchconfig

![flowchart](enforce-aws-ecs-asg-launchconfig.png)

This tool is useful to ensure that all EC2 instances in a ECS cluster backed up by a ASG share the launch configuration defined in the ASG. This tool doesn't work with launch templates. ECS EC2 Container Instances will be drained. EC2 Instances will be terminated (after they have been drained).

Get:

Grab binary distribution from [releases tab](https://git.autodesk.com/t-villa/go-awsecs/releases). Or.

```
go get -v -u git.autodesk.com/t-villa/go-awsecs/cmd/enforce-aws-ecs-asg-launchconfig
```

Use:

```
enforce-aws-ecs-asg-launchconfig -h
Usage of enforce-aws-ecs-asg-launchconfig:
  -asg string
    	asg name
  -cluster string
    	cluster name
  -profile string
    	profile name
  -region string
    	region name
```

Example:

```
enforce-aws-ecs-asg-launchconfig \
  -asg myasgname \
  -cluster myclustername
# default timeout for the operation is 15 minutes
```

----

1. https://unix.stackexchange.com/a/111557/19393
2. https://docs.aws.amazon.com/AmazonECS/latest/APIReference/API_Deployment.html
3. To Be Compared
