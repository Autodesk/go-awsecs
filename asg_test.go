package awsecs

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
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
	instances, name, err := listASGInstaces(&mockAutoScalingClient{}, "")
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
