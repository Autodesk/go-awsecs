package awsecs

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/cenkalti/backoff"
	"log"
	"strings"
	"sync"
)

func listASGInstaces(ASAPI autoscalingiface.AutoScalingAPI, asgName string) ([]*autoscaling.Instance, *string, error) {
	output, err := ASAPI.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{aws.String(asgName)},
	})
	if err != nil {
		return []*autoscaling.Instance{}, nil, err
	}

	for _, autoScalingGroup := range output.AutoScalingGroups {
		return autoScalingGroup.Instances, autoScalingGroup.LaunchConfigurationName, nil
	}

	return []*autoscaling.Instance{}, nil, errors.New("asg not found")
}

func needReplacement(expectedLaunchConfig string, instance autoscaling.Instance) bool {
	if instance.LaunchConfigurationName == nil {
		return true
	}
	if *instance.LaunchConfigurationName != expectedLaunchConfig {
		return true
	}
	return false
}

func filterInstancesToReplace(expectedLaunchConfig *string, instances []*autoscaling.Instance) (filtered []autoscaling.Instance) {
	if expectedLaunchConfig == nil {
		return
	}
	for _, instance := range instances {
		if instance != nil {
			if needReplacement(*expectedLaunchConfig, *instance) {
				filtered = append(filtered, *instance)
			}
		}
	}
	return
}

type ecsEC2Instance struct {
	ec2InstanceID          string
	ecsContainerInstanceID string
}

func containerInstanceArnsToContainerInstanceIds(input []*string) (output []*string) {
	for _, arn := range input {
		output = append(output, aws.String(containerInstanceArnToContainerInstanceID(*arn)))
	}
	return
}

func containerInstanceArnToContainerInstanceID(input string) (output string) {
	parts := strings.Split(input, "/")
	output = strings.Join(parts[len(parts)-1:], "")
	return
}

func instancesToContainerInstances(ECSAPI ecs.ECS, instances []autoscaling.Instance, clusterName string) ([]ecsEC2Instance, error) {
	var ecsEC2Instances []ecsEC2Instance
	var describeContainerInstancesInputs []*ecs.DescribeContainerInstancesInput
	err := ECSAPI.ListContainerInstancesPages(
		&ecs.ListContainerInstancesInput{Cluster: aws.String(clusterName)},
		func(page *ecs.ListContainerInstancesOutput, lastPage bool) bool {
			describeContainerInstancesInputs = append(describeContainerInstancesInputs, &ecs.DescribeContainerInstancesInput{
				Cluster:            aws.String(clusterName),
				ContainerInstances: containerInstanceArnsToContainerInstanceIds(page.ContainerInstanceArns),
			})
			return true
		})
	if err != nil {
		return []ecsEC2Instance{}, err
	}
	for _, input := range describeContainerInstancesInputs {
		output, err := ECSAPI.DescribeContainerInstances(input)
		if err != nil {
			return []ecsEC2Instance{}, err
		}
		for _, instance := range instances {
			for _, containerInstance := range output.ContainerInstances {
				if *containerInstance.Ec2InstanceId == *instance.InstanceId {
					ecsEC2Instances = append(ecsEC2Instances, ecsEC2Instance{
						ec2InstanceID:          *instance.InstanceId,
						ecsContainerInstanceID: containerInstanceArnToContainerInstanceID(*containerInstance.ContainerInstanceArn),
					})
					goto gotobreak
				}
			}
		gotobreak:
		}
	}
	return ecsEC2Instances, nil
}

func detachAndDrain(ASAPI autoscaling.AutoScaling, ECSAPI ecs.ECS, instance ecsEC2Instance, asgName, clusterName string) error {
	output, err := ASAPI.DetachInstances(&autoscaling.DetachInstancesInput{
		AutoScalingGroupName:           aws.String(asgName),
		InstanceIds:                    []*string{aws.String(instance.ec2InstanceID)},
		ShouldDecrementDesiredCapacity: aws.Bool(false),
	})

	if err != nil {
		return fmt.Errorf("%v %v", instance, err)
	}

	for _, activity := range output.Activities {
		log.Printf("%v %v", instance, *activity.Description)
	}

	_, err = ECSAPI.UpdateContainerInstancesState(&ecs.UpdateContainerInstancesStateInput{
		Cluster:            aws.String(clusterName),
		ContainerInstances: []*string{aws.String(instance.ecsContainerInstanceID)},
		Status:             aws.String("DRAINING"),
	})

	reAttach := func() {
		_, err2 := ASAPI.AttachInstances(&autoscaling.AttachInstancesInput{
			AutoScalingGroupName: aws.String(asgName),
			InstanceIds:          []*string{aws.String(instance.ec2InstanceID)},
		})
		if err2 != nil {
			log.Printf("[ACTIONABLE ACTION REQUIRED] instance re-attachment failed!")
			log.Printf("%v %v", instance, err2)
		}
	}

	if err != nil {
		reAttach()
		return fmt.Errorf("%v %v", instance, err)
	}

	operation := func() error {
		err := drainingContainerInstanceIsDrained(ECSAPI, clusterName, instance.ecsContainerInstanceID)
		if err != nil {
			log.Printf("%v %v", instance, err)
		}
		return err
	}

	err = backoff.Retry(operation, backoff.NewExponentialBackOff())
	if err != nil {
		reAttach()
		log.Printf("[ACTIONABLE ACTION REQUIRED] instance left in DRAINING status!")
		return fmt.Errorf("%v %v", instance, err)
	}

	return nil
}

func drainingContainerInstanceIsDrained(ECSAPI ecs.ECS, clusterName, containerInstanceID string) error {
	output, err := ECSAPI.DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(clusterName),
		ContainerInstances: []*string{aws.String(containerInstanceID)},
	})
	if err != nil {
		return err
	}
	for _, containerInstance := range output.ContainerInstances {
		if *containerInstance.Status != "DRAINING" {
			return backoff.Permanent(errors.New("the instance should be DRAINING but is not"))
		}
		if *containerInstance.RunningTasksCount != 0 {
			return errors.New("container instance still DRAINING")
		}
		return nil
	}
	return backoff.Permanent(errors.New("container instance not found"))
}

func drainAll(ASAPI autoscaling.AutoScaling, ECSAPI ecs.ECS, EC2API ec2.EC2, instances []ecsEC2Instance, asgName, clusterName string) error {
	errors := make([]error, len(instances))
	var wg sync.WaitGroup
	for i, instance := range instances {
		wg.Add(1)
		go func(thatInstance ecsEC2Instance, index int) {
			defer wg.Done()
			errors[index] = detachAndDrain(ASAPI, ECSAPI, thatInstance, asgName, clusterName)
			if errors[index] == nil {
				_, err := EC2API.TerminateInstances(&ec2.TerminateInstancesInput{
					InstanceIds: []*string{
						aws.String(thatInstance.ec2InstanceID),
					},
				})
				errors[index] = err
			}
		}(instance, i)
	}
	wg.Wait()
	var onlyErrors []error
	for _, err := range errors {
		if err != nil {
			onlyErrors = append(onlyErrors, err)
		}
	}
	if len(onlyErrors) > 0 {
		return fmt.Errorf("%v", onlyErrors)
	}
	return nil
}

func enforceLaunchConfig(ECSAPI ecs.ECS, ASAPI autoscaling.AutoScaling, EC2API ec2.EC2, asgName, clusterName string, bo backoff.BackOff) error {
	asgInstances, expectedLaunchConfig, err := listASGInstaces(&ASAPI, asgName)
	if err != nil {
		return err
	}
	instances, err := instancesToContainerInstances(ECSAPI, filterInstancesToReplace(expectedLaunchConfig, asgInstances), clusterName)
	if err != nil {
		return err
	}
	return drainAll(ASAPI, ECSAPI, EC2API, instances, asgName, clusterName)
}

// EnforceLaunchConfig encapsulates the attributes of a LaunchConfig enforcement
type EnforceLaunchConfig struct {
	ECSAPI         ecs.ECS
	ASAPI          autoscaling.AutoScaling
	EC2API         ec2.EC2
	ASGName        string
	ECSClusterName string
	BackOff        backoff.BackOff
}

// Apply the LaunchConfig enforcement
func (e *EnforceLaunchConfig) Apply() error {
	return enforceLaunchConfig(e.ECSAPI, e.ASAPI, e.EC2API, e.ASGName, e.ECSClusterName, e.BackOff)
}
