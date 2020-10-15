package awsecs

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

const TaskRoleKnockoutValue = "None"

func alterTaskRole(copy ecs.RegisterTaskDefinitionInput, taskRoleArn string) ecs.RegisterTaskDefinitionInput {
	obj, err := json.Marshal(copy)
	if err != nil {
		panic(err)
	}
	copyClone := ecs.RegisterTaskDefinitionInput{}
	err = json.Unmarshal(obj, &copyClone)
	if err != nil {
		panic(err)
	}
	if taskRoleArn != "" {
		copyClone.TaskRoleArn = aws.String(taskRoleArn)
	}
	if taskRoleArn == TaskRoleKnockoutValue {
		copyClone.TaskRoleArn = nil
	}
	return copyClone
}
