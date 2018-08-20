package awsecs

import (
	"errors"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/cenkalti/backoff"
	"log"
)

var (
	// ErrOtherThanPrimaryDeploymentFound service update didn't complete
	ErrOtherThanPrimaryDeploymentFound = errors.New("other than PRIMARY deployment found")
	// ErrServiceNotFound trying to update a service that doesn't exist
	ErrServiceNotFound = errors.New("the service does not exist")
	// ErrServiceDeletedAfterUpdate service was updated and then deleted elsewhere
	ErrServiceDeletedAfterUpdate = backoff.Permanent(errors.New("the service was deleted after the update"))
)

var (
	errNoPrimaryDeployment = backoff.Permanent(errors.New("no PRIMARY deployment"))
)

func copy(input *ecs.TaskDefinition) (output *ecs.RegisterTaskDefinitionInput) {
	output = &ecs.RegisterTaskDefinitionInput{}
	output.ContainerDefinitions = input.ContainerDefinitions
	output.Cpu = input.Cpu
	output.ExecutionRoleArn = input.ExecutionRoleArn
	output.Family = input.Family
	output.Memory = input.Memory
	output.NetworkMode = input.NetworkMode
	output.PlacementConstraints = input.PlacementConstraints
	output.RequiresCompatibilities = input.RequiresCompatibilities
	output.TaskRoleArn = input.TaskRoleArn
	output.Volumes = input.Volumes
	return
}

func alterImages(copy *ecs.RegisterTaskDefinitionInput, kvs map[string]string) {
	for k, v := range kvs {
		for _, containerDefinition := range copy.ContainerDefinitions {
			if *containerDefinition.Name == k {
				containerDefinition.Image = &v
			}
		}
	}
}

func alterEnvironments(copy *ecs.RegisterTaskDefinitionInput, kvs map[string]map[string]string) {
	for k, v := range kvs {
		for _, containerDefinition := range copy.ContainerDefinitions {
			if *containerDefinition.Name == k {
				alterEnvironment(containerDefinition, v)
			}
		}
	}
}

func alterEnvironment(copy *ecs.ContainerDefinition, kvs map[string]string) {
	for k, v := range kvs {
		for _, environment := range copy.Environment {
			if *environment.Name == k {
				environment.Value = &v
				break
			}
			copy.Environment = append(copy.Environment, &ecs.KeyValuePair{Name: &k, Value: &v})
		}
	}
}

func copyTaskDef(api ecs.ECS, taskdef *string, kvs map[string]string, kvs2 map[string]map[string]string) (arn *string, err error) {
	var output *ecs.DescribeTaskDefinitionOutput
	var new *ecs.RegisterTaskDefinitionOutput
	output, err = api.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{TaskDefinition: taskdef})
	if err != nil {
		return
	}
	copy := copy(output.TaskDefinition)
	alterImages(copy, kvs)
	alterEnvironments(copy, kvs2)
	new, err = api.RegisterTaskDefinition(copy)
	if err != nil {
		return
	}
	arn = new.TaskDefinition.TaskDefinitionArn
	return
}

func alterService(api ecs.ECS, cluster, service *string, kvs map[string]string, kvs2 map[string]map[string]string, desiredCount *int64) (ecsService *ecs.Service, err error) {
	var output *ecs.DescribeServicesOutput
	var output2 *ecs.UpdateServiceOutput
	if output, err = api.DescribeServices(&ecs.DescribeServicesInput{Cluster: cluster, Services: []*string{service}}); err != nil {
		return
	}
	err = ErrServiceNotFound
	for _, svc := range output.Services {
		var newTd *string
		if newTd, err = copyTaskDef(api, svc.TaskDefinition, kvs, kvs2); err != nil {
			return
		}
		if desiredCount == nil {
			desiredCount = svc.DesiredCount
		}
		if output2, err = api.UpdateService(&ecs.UpdateServiceInput{Cluster: cluster, Service: service, TaskDefinition: newTd, DesiredCount: desiredCount}); err != nil {
			return
		}
		ecsService = output2.Service
	}
	return
}

func validateDeployment(api ecs.ECS, ecsService *ecs.Service) error {
	for _, ecsDeployment := range ecsService.Deployments {
		if *ecsDeployment.Status == "PRIMARY" {
			output, err := api.DescribeServices(&ecs.DescribeServicesInput{Cluster: ecsService.ClusterArn, Services: []*string{ecsService.ServiceName}})
			if err != nil {
				return err
			}
			for _, svc := range output.Services {
				for _, deployment := range svc.Deployments {
					if *deployment.Id != *ecsDeployment.Id {
						return ErrOtherThanPrimaryDeploymentFound
					}
				}
				for _, deployment := range svc.Deployments {
					if *deployment.Id == *ecsDeployment.Id {
						return nil
					}
				}
			}
			return ErrServiceDeletedAfterUpdate
		}
	}
	return errNoPrimaryDeployment
}

func alterServiceValidateDeployment(api ecs.ECS, cluster, service *string, kvs map[string]string, kvs2 map[string]map[string]string, desiredCount *int64, bo backoff.BackOff) (err error) {
	var svc *ecs.Service
	if svc, err = alterService(api, cluster, service, kvs, kvs2, desiredCount); err != nil {
		return
	}

	operation := func() error {
		err := validateDeployment(api, svc)
		if err != nil {
			log.Print(err)
		}
		return err
	}

	return backoff.Retry(operation, bo)
}

// ECSServiceUpdate encapsulates the attributes of an ECS service update
type ECSServiceUpdate struct {
	API          ecs.ECS                      // ECS Api
	Cluster      string                       // Cluster which the service is deployed to
	Service      string                       // Name of the service
	Image        map[string]string            // Map of container names and images
	Environment  map[string]map[string]string // Map of container names environment variable name and value
	DesiredCount *int64                       // If nil the service desired count is not altered
	BackOff      backoff.BackOff              // BackOff strategy to use when validating the update
}

// Apply the ECS Service Update
func (e *ECSServiceUpdate) Apply() error {
	return alterServiceValidateDeployment(e.API, &e.Cluster, &e.Service, e.Image, e.Environment, e.DesiredCount, e.BackOff)
}
