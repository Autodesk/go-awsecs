package awsecs

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

const TaskRoleKnockoutValue = "None"

func alterTaskRole(copy ecs.RegisterTaskDefinitionInput, taskRoleArn string) ecs.RegisterTaskDefinitionInput {
	obj := panicMarshal(copy)
	copyClone := ecs.RegisterTaskDefinitionInput{}
	panicUnmarshal(obj, &copyClone)
	if taskRoleArn != "" {
		copyClone.TaskRoleArn = aws.String(taskRoleArn)
	}
	if taskRoleArn == TaskRoleKnockoutValue {
		copyClone.TaskRoleArn = nil
	}
	return copyClone
}
