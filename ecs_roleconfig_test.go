package awsecs

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"reflect"
	"testing"
)

func TestAlterTaskRole(t *testing.T) {
	type args struct {
		copy        ecs.RegisterTaskDefinitionInput
		taskRoleArn string
	}
	tests := []struct {
		name string
		args args
		want ecs.RegisterTaskDefinitionInput
	}{
		{
			name: "None test",
			args: args{
				ecs.RegisterTaskDefinitionInput{
					TaskRoleArn: aws.String("something")},
				"None",
			},
			want: ecs.RegisterTaskDefinitionInput{},
		},
		{
			name: "Set value test",
			args: args{
				ecs.RegisterTaskDefinitionInput{},
				"taskRoleArn",
			},
			want: ecs.RegisterTaskDefinitionInput{
				TaskRoleArn: aws.String("taskRoleArn"),
			},
		},
		{
			name: "Keep value test",
			args: args{
				ecs.RegisterTaskDefinitionInput{
					TaskRoleArn: aws.String("keepTaskRoleArn"),
				},
				"",
			},
			want: ecs.RegisterTaskDefinitionInput{
				TaskRoleArn: aws.String("keepTaskRoleArn"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := alterTaskRole(tt.args.copy, tt.args.taskRoleArn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("alterTaskRole() = %v, want %v", got, tt.want)
			}
		})
	}
}
