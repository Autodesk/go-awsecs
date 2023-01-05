package awsecs

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/ecs"
	"testing"
)

type mockAutoScalingClient struct {
	autoscalingiface.AutoScalingAPI
}

func (m *mockAutoScalingClient) DescribeAutoScalingGroups(input *autoscaling.DescribeAutoScalingGroupsInput) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	name := ""
	return &autoscaling.DescribeAutoScalingGroupsOutput{
		AutoScalingGroups: []*autoscaling.Group{
			&autoscaling.Group{
				Instances: []*autoscaling.Instance{
					// TODO
				},
				LaunchConfigurationName: &name,
			},
		},
	}, nil
}

func TestListASGInstaces(t *testing.T) {
	instances, name, err := listASGInstances(&mockAutoScalingClient{}, "")
	if len(instances) != 0 {
		t.Errorf("unexpected")
	}
	if *name != "" {
		t.Errorf("unexpected")
	}
	if err != nil {
		t.Errorf("unexpected")
	}
}

func TestNeedReplacement(t *testing.T) {
	name := ""
	replace := needReplacement("", autoscaling.Instance{LaunchConfigurationName: &name})
	if replace {
		t.Errorf("unexpected")
	}
}

func TestFilterInstancesToReplace(t *testing.T) {
	name := ""
	dontReplace := autoscaling.Instance{LaunchConfigurationName: &name}
	replace := autoscaling.Instance{}
	listToReplace := filterInstancesToReplace(&name, []*autoscaling.Instance{&dontReplace, &replace})
	if listToReplace[0] != replace {
		t.Errorf("unexpected")
	}
}

func TestCheckDrainingContainerInstance(t *testing.T) {
	type args struct {
		containerInstance   *ecs.ContainerInstance
		parsedArn           arn.ARN
		containerInstanceID string
	}
	tests := []struct {
		name    string
		wantErr bool
		args    args
	}{
		{
			name:    "Found container instance ACTIVE",
			wantErr: true,
			args: args{
				containerInstance: &ecs.ContainerInstance{
					Status: aws.String(ecs.ContainerInstanceStatusActive),
				},
				parsedArn: arn.ARN{
					Resource: "container-instance/container_instance_ID",
				},
				containerInstanceID: "container_instance_ID",
			},
		},
		{
			name:    "Found container instance DRAINING and running tasks",
			wantErr: true,
			args: args{
				containerInstance: &ecs.ContainerInstance{
					Status:            aws.String(ecs.ContainerInstanceStatusDraining),
					RunningTasksCount: aws.Int64(10),
				},
				parsedArn: arn.ARN{
					Resource: "container-instance/container_instance_ID",
				},
				containerInstanceID: "container_instance_ID",
			},
		},
		{
			name:    "Found container instance DRAINING and no longer running tasks",
			wantErr: false,
			args: args{
				containerInstance: &ecs.ContainerInstance{
					Status:            aws.String(ecs.ContainerInstanceStatusDraining),
					RunningTasksCount: aws.Int64(0),
				},
				parsedArn: arn.ARN{
					Resource: "container-instance/container_instance_ID",
				},
				containerInstanceID: "container_instance_ID",
			},
		},
		{
			name:    "Not matching container instance ID",
			wantErr: true,
			args: args{
				containerInstance: &ecs.ContainerInstance{},
				parsedArn: arn.ARN{
					Resource: "container-instance/another_container_instance_ID",
				},
				containerInstanceID: "container_instance_ID",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkDrainingContainerInstance(tt.args.containerInstance, tt.args.parsedArn, tt.args.containerInstanceID); (err != nil) != tt.wantErr {
				t.Errorf("checkDrainingContainerInstance() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
