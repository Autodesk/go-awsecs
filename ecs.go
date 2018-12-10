package awsecs

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/cenkalti/backoff"
	"log"
)

var (
	// EnvKnockOutValue value used to knock off environment variables
	EnvKnockOutValue = ""
	// ErrDeploymentChangedElsewhere the deployment was changed elsewhere
	ErrDeploymentChangedElsewhere = backoff.Permanent(errors.New("the deployment was changed elsewhere"))
	// ErrOtherThanPrimaryDeploymentFound service update didn't complete
	ErrOtherThanPrimaryDeploymentFound = errors.New("other than PRIMARY deployment found")
	// ErrNotRunningDesiredCount service update completed but number of containers not matching desired count
	ErrNotRunningDesiredCount = errors.New("not running the desired count")
	// ErrServiceNotFound trying to update a service that doesn't exist
	ErrServiceNotFound = errors.New("the service does not exist")
	// ErrServiceDeletedAfterUpdate service was updated and then deleted elsewhere
	ErrServiceDeletedAfterUpdate = backoff.Permanent(errors.New("the service was deleted after the update"))
)

var (
	errNoPrimaryDeployment = backoff.Permanent(errors.New("no PRIMARY deployment"))
)

func copy(input ecs.TaskDefinition) ecs.RegisterTaskDefinitionInput {
	output := ecs.RegisterTaskDefinitionInput{}
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
	return output
}

func alterImages(copy ecs.RegisterTaskDefinitionInput, imageMap map[string]string) ecs.RegisterTaskDefinitionInput {
	for name, image := range imageMap {
		for _, containerDefinition := range copy.ContainerDefinitions {
			if *containerDefinition.Name == name {
				containerDefinition.Image = aws.String(image)
			}
		}
	}
	return copy
}

func alterEnvironments(copy ecs.RegisterTaskDefinitionInput, envMaps map[string]map[string]string) ecs.RegisterTaskDefinitionInput {
	for name, envMap := range envMaps {
		for i, containerDefinition := range copy.ContainerDefinitions {
			if *containerDefinition.Name == name {
				new := alterEnvironment(*containerDefinition, envMap)
				copy.ContainerDefinitions[i] = &new
			}
		}
	}
	return copy
}

func alterEnvironment(copy ecs.ContainerDefinition, envMap map[string]string) ecs.ContainerDefinition {
	for name, value := range envMap {
		i := 0
		found := false
		for i < len(copy.Environment) {
			environment := copy.Environment[i]
			if *environment.Name == name && value == EnvKnockOutValue {
				copy.Environment = append(copy.Environment[:i], copy.Environment[i+1:]...)
				found = true
				i--
			} else if *environment.Name == name {
				environment.Value = aws.String(value)
				found = true
			}
			i++
		}
		if !found && value != EnvKnockOutValue {
			copy.Environment = append(copy.Environment, &ecs.KeyValuePair{Name: aws.String(name), Value: aws.String(value)})
		}
	}
	return copy
}

func copyTaskDef(api ecs.ECS, taskdef string, imageMap map[string]string, envMaps map[string]map[string]string) (string, error) {
	output, err := api.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{TaskDefinition: aws.String(taskdef)})
	if err != nil {
		return "", err
	}
	copy := alterEnvironments(alterImages(copy(*output.TaskDefinition), imageMap), envMaps)
	new, err := api.RegisterTaskDefinition(&copy)
	if err != nil {
		return "", err
	}
	arn := new.TaskDefinition.TaskDefinitionArn
	return *arn, nil
}

func alterService(api ecs.ECS, cluster, service string, imageMap map[string]string, envMaps map[string]map[string]string, desiredCount *int64) (ecs.Service, ecs.Service, error) {
	output, err := api.DescribeServices(&ecs.DescribeServicesInput{Cluster: aws.String(cluster), Services: []*string{aws.String(service)}})
	if err != nil {
		return ecs.Service{}, ecs.Service{}, err
	}
	for _, svc := range output.Services {
		newTd, err := copyTaskDef(api, *svc.TaskDefinition, imageMap, envMaps)
		if err != nil {
			return *svc, ecs.Service{}, err
		}
		if desiredCount == nil {
			desiredCount = svc.DesiredCount
		}
		updated, err := api.UpdateService(&ecs.UpdateServiceInput{Cluster: aws.String(cluster), Service: aws.String(service), TaskDefinition: aws.String(newTd), DesiredCount: desiredCount})
		if err != nil {
			return *svc, ecs.Service{}, err
		}
		return *svc, *updated.Service, nil
	}
	return ecs.Service{}, ecs.Service{}, ErrServiceNotFound
}

func validateDeployment(api ecs.ECS, ecsService ecs.Service) error {
	for _, ecsDeployment := range ecsService.Deployments {
		if *ecsDeployment.Status == "PRIMARY" {
			output, err := api.DescribeServices(&ecs.DescribeServicesInput{Cluster: ecsService.ClusterArn, Services: []*string{ecsService.ServiceName}})
			if err != nil {
				return err
			}
			for _, svc := range output.Services {
				for _, deployment := range svc.Deployments {
					if *deployment.Status == "PRIMARY" && *deployment.Id != *ecsDeployment.Id {
						return ErrDeploymentChangedElsewhere
					}
					if *deployment.Id != *ecsDeployment.Id {
						return ErrOtherThanPrimaryDeploymentFound
					}
				}
				for _, deployment := range svc.Deployments {
					if *deployment.Id == *ecsDeployment.Id {
						if *svc.RunningCount < *svc.DesiredCount {
							return ErrNotRunningDesiredCount
						}
						return nil
					}
				}
			}
			return ErrServiceDeletedAfterUpdate
		}
	}
	return errNoPrimaryDeployment
}

func alterServiceValidateDeployment(api ecs.ECS, cluster, service string, imageMap map[string]string, envMaps map[string]map[string]string, desiredCount *int64, bo backoff.BackOff) (ecs.Service, error) {
	oldsvc, newsvc, err := alterService(api, cluster, service, imageMap, envMaps, desiredCount)
	if err != nil {
		return oldsvc, err
	}
	operation := func() error {
		err := validateDeployment(api, newsvc)
		if err != nil {
			log.Print(err)
		}
		return err
	}
	return oldsvc, backoff.Retry(operation, bo)
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
	return alterServiceOrValidatedRollBack(e.API, e.Cluster, e.Service, e.Image, e.Environment, e.DesiredCount, e.BackOff)
}
