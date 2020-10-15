package awsecs

import (
	"encoding/json"
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/cenkalti/backoff"
	"log"
	"reflect"
)

var (
	// EnvKnockOutValue value used to knock off environment variables
	EnvKnockOutValue = ""
	// ErrDeploymentChangedElsewhere the deployment was changed elsewhere
	ErrDeploymentChangedElsewhere = errors.New("the deployment was changed elsewhere")
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

func copyTd(input ecs.TaskDefinition, tags []*ecs.Tag) ecs.RegisterTaskDefinitionInput {
	obj, err := json.Marshal(input)
	if err != nil {
		panic(err)
	}
	inputClone := ecs.TaskDefinition{}
	err = json.Unmarshal(obj, &inputClone)
	if err != nil {
		panic(err)
	}
	output := ecs.RegisterTaskDefinitionInput{}
	// TODO: replace with reflection
	output.ContainerDefinitions = inputClone.ContainerDefinitions
	output.Cpu = inputClone.Cpu
	output.ExecutionRoleArn = inputClone.ExecutionRoleArn
	output.Family = inputClone.Family
	output.InferenceAccelerators = inputClone.InferenceAccelerators
	output.IpcMode = inputClone.IpcMode
	output.Memory = inputClone.Memory
	output.NetworkMode = inputClone.NetworkMode
	output.PidMode = inputClone.PidMode
	output.PlacementConstraints = inputClone.PlacementConstraints
	output.ProxyConfiguration = inputClone.ProxyConfiguration
	output.RequiresCompatibilities = inputClone.RequiresCompatibilities
	output.TaskRoleArn = inputClone.TaskRoleArn
	output.Volumes = inputClone.Volumes
	// can't be replaced with reflection
	output.Tags = tags
	return output
}

func alterImages(copy ecs.RegisterTaskDefinitionInput, imageMap map[string]string) ecs.RegisterTaskDefinitionInput {
	obj, err := json.Marshal(copy)
	if err != nil {
		panic(err)
	}
	copyClone := ecs.RegisterTaskDefinitionInput{}
	err = json.Unmarshal(obj, &copyClone)
	if err != nil {
		panic(err)
	}
	for name, image := range imageMap {
		for _, containerDefinition := range copyClone.ContainerDefinitions {
			if *containerDefinition.Name == name {
				containerDefinition.Image = aws.String(image)
			}
		}
	}
	return copyClone
}

func alterEnvironments(copy ecs.RegisterTaskDefinitionInput, envMaps map[string]map[string]string) ecs.RegisterTaskDefinitionInput {
	obj, err := json.Marshal(copy)
	if err != nil {
		panic(err)
	}
	copyClone := ecs.RegisterTaskDefinitionInput{}
	err = json.Unmarshal(obj, &copyClone)
	if err != nil {
		panic(err)
	}
	for name, envMap := range envMaps {
		for i, containerDefinition := range copyClone.ContainerDefinitions {
			if *containerDefinition.Name == name {
				altered := alterEnvironment(*containerDefinition, envMap)
				copyClone.ContainerDefinitions[i] = &altered
			}
		}
	}
	return copyClone
}

func alterSecrets(copy ecs.RegisterTaskDefinitionInput, secretMaps map[string]map[string]string) ecs.RegisterTaskDefinitionInput {
	obj, err := json.Marshal(copy)
	if err != nil {
		panic(err)
	}
	copyClone := ecs.RegisterTaskDefinitionInput{}
	err = json.Unmarshal(obj, &copyClone)
	if err != nil {
		panic(err)
	}
	for name, secretMap := range secretMaps {
		for i, containerDefinition := range copyClone.ContainerDefinitions {
			if *containerDefinition.Name == name {
				altered := alterSecret(*containerDefinition, secretMap)
				copyClone.ContainerDefinitions[i] = &altered
			}
		}
	}
	return copyClone
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

func alterSecret(copy ecs.ContainerDefinition, secretMap map[string]string) ecs.ContainerDefinition {
	for name, valueFrom := range secretMap {
		i := 0
		found := false
		for i < len(copy.Secrets) {
			secret := copy.Secrets[i]
			if *secret.Name == name && valueFrom == EnvKnockOutValue {
				copy.Secrets = append(copy.Secrets[:i], copy.Secrets[i+1:]...)
				found = true
				i--
			} else if *secret.Name == name {
				secret.ValueFrom = aws.String(valueFrom)
				found = true
			}
			i++
		}
		if !found && valueFrom != EnvKnockOutValue {
			copy.Secrets = append(copy.Secrets, &ecs.Secret{Name: aws.String(name), ValueFrom: aws.String(valueFrom)})
		}
	}
	return copy
}

func copyTaskDef(api ecs.ECS, taskdef string, imageMap map[string]string, envMaps map[string]map[string]string, secretMaps map[string]map[string]string, logopts map[string]map[string]map[string]string, logsecrets map[string]map[string]map[string]string, taskRole string) (string, error) {
	output, err := api.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{TaskDefinition: aws.String(taskdef)})
	if err != nil {
		return "", err
	}

	asRegisterTaskDefinitionInput := copyTd(*output.TaskDefinition, output.Tags)
	tdCopy := alterImages(asRegisterTaskDefinitionInput, imageMap)
	tdCopy = alterEnvironments(tdCopy, envMaps)
	tdCopy = alterSecrets(tdCopy, secretMaps)
	tdCopy = alterLogConfigurations(tdCopy, logopts, logsecrets)
	tdCopy = alterTaskRole(tdCopy, taskRole)

	if reflect.DeepEqual(asRegisterTaskDefinitionInput, tdCopy) {
		return *output.TaskDefinition.TaskDefinitionArn, nil
	}
	tdNew, err := api.RegisterTaskDefinition(&tdCopy)
	if err != nil {
		return "", err
	}
	arn := tdNew.TaskDefinition.TaskDefinitionArn
	return *arn, nil
}

func alterService(api ecs.ECS, cluster, service string, imageMap map[string]string, envMaps map[string]map[string]string, secretMaps map[string]map[string]string, logopts map[string]map[string]map[string]string, logsecrets map[string]map[string]map[string]string, taskRole string, desiredCount *int64, taskdef string) (ecs.Service, ecs.Service, error) {
	output, err := api.DescribeServices(&ecs.DescribeServicesInput{Cluster: aws.String(cluster), Services: []*string{aws.String(service)}})
	if err != nil {
		return ecs.Service{}, ecs.Service{}, err
	}
	for _, svc := range output.Services {
		srcTaskDef := svc.TaskDefinition
		if taskdef != "" {
			srcTaskDef = &taskdef
		}
		newTd, err := copyTaskDef(api, *srcTaskDef, imageMap, envMaps, secretMaps, logopts, logsecrets, taskRole)
		if err != nil {
			return *svc, ecs.Service{}, err
		}
		if desiredCount == nil {
			desiredCount = svc.DesiredCount
		}
		updated, err := api.UpdateService(&ecs.UpdateServiceInput{Cluster: aws.String(cluster), Service: aws.String(service), TaskDefinition: aws.String(newTd), DesiredCount: desiredCount, ForceNewDeployment: aws.Bool(true)})
		if err != nil {
			return *svc, ecs.Service{}, err
		}
		return *svc, *updated.Service, nil
	}
	return ecs.Service{}, ecs.Service{}, ErrServiceNotFound
}

func validateDeployment(api ecs.ECS, ecsService ecs.Service, bo backoff.BackOff) error {
	for _, ecsDeployment := range ecsService.Deployments {
		if *ecsDeployment.Status == "PRIMARY" {

			var output *ecs.DescribeServicesOutput
			var err error

			operation := func() error {
				output, err = api.DescribeServices(&ecs.DescribeServicesInput{Cluster: ecsService.ClusterArn, Services: []*string{ecsService.ServiceName}})
				if err != nil {
					return err
				}
				for _, svc := range output.Services {
					for _, deployment := range svc.Deployments {
						if *deployment.Status == "PRIMARY" && *deployment.Id != *ecsDeployment.Id {
							return ErrDeploymentChangedElsewhere
						}
					}
				}
				return nil
			}

			err = backoff.Retry(operation, backoff.WithMaxRetries(bo, 5))
			if err == ErrDeploymentChangedElsewhere {
				return backoff.Permanent(err)
			}

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

func alterServiceValidateDeployment(api ecs.ECS, cluster, service string, imageMap map[string]string, envMaps map[string]map[string]string, secretMaps map[string]map[string]string, logopts map[string]map[string]map[string]string, logsecrets map[string]map[string]map[string]string, taskRole string, desiredCount *int64, taskdef string, bo backoff.BackOff) (ecs.Service, error) {
	oldsvc, newsvc, err := alterService(api, cluster, service, imageMap, envMaps, secretMaps, logopts, logsecrets, taskRole, desiredCount, taskdef)
	if err != nil {
		return oldsvc, err
	}
	var prevErr error
	operation := func() error {
		err := validateDeployment(api, newsvc, bo)
		if err != prevErr && err != nil {
			prevErr = err
			log.Print(err)
		}
		return err
	}
	return oldsvc, backoff.Retry(operation, bo)
}

// ECSServiceUpdate encapsulates the attributes of an ECS service update
type ECSServiceUpdate struct {
	API              ecs.ECS                                 // ECS Api
	Cluster          string                                  // Cluster which the service is deployed to
	Service          string                                  // Name of the service
	Image            map[string]string                       // Map of container names and images
	Environment      map[string]map[string]string            // Map of container names environment variable name and value
	Secrets          map[string]map[string]string            // Map of container names environment variable name and valueFrom
	LogDriverOptions map[string]map[string]map[string]string // Map of container names log driver name log driver option and value
	LogDriverSecrets map[string]map[string]map[string]string // Map of container names log driver name log driver secret and valueFrom
	TaskRole         string                                  // Task IAM Role if TaskRoleKnockoutValue used, it is cleared
	DesiredCount     *int64                                  // If nil the service desired count is not altered
	BackOff          backoff.BackOff                         // BackOff strategy to use when validating the update
	Taskdef          string                                  // If non empty used as base task definition instead of the current task definition
}

// Apply the ECS Service Update
func (e *ECSServiceUpdate) Apply() error {
	return alterServiceOrValidatedRollBack(e.API, e.Cluster, e.Service, e.Image, e.Environment, e.Secrets, e.LogDriverOptions, e.LogDriverSecrets, e.TaskRole, e.DesiredCount, e.Taskdef, e.BackOff)
}
