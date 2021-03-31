package awsecs

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/elbv2/elbv2iface"
	"github.com/cenkalti/backoff"
	"log"
	"reflect"
	"strings"
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
	// ErrWaitingForDrainingState the service doesn't have any target which transitioned to draining state
	ErrWaitingForDrainingState = errors.New("waiting for draining state")
	// ErrInvalidWaitUntil received an invalid wait until
	ErrInvalidWaitUntil = errors.New("invalid wait until received")
	// ErrServiceDeletedAfterUpdate service was updated and then deleted elsewhere
	ErrServiceDeletedAfterUpdate = backoff.Permanent(errors.New("the service was deleted after the update"))
	// ErrContainerInstanceNotFound the container instance was removed from the cluster elsewhere
	ErrContainerInstanceNotFound = backoff.Permanent(errors.New("container instance not found"))
	// ErrLoadBalancerNotConfigured the service doesn't have a load balancer configured
	ErrLoadBalancerNotConfigured = backoff.Permanent(errors.New("the service was deleted after the update"))
)

var (
	errNoPrimaryDeployment = backoff.Permanent(errors.New("no PRIMARY deployment"))
)

func copyTd(input ecs.TaskDefinition, tags []*ecs.Tag) ecs.RegisterTaskDefinitionInput {
	obj := panicMarshal(input)
	inputClone := ecs.TaskDefinition{}
	panicUnmarshal(obj, &inputClone)
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
	obj := panicMarshal(copy)
	copyClone := ecs.RegisterTaskDefinitionInput{}
	panicUnmarshal(obj, &copyClone)
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
	obj := panicMarshal(copy)
	copyClone := ecs.RegisterTaskDefinitionInput{}
	panicUnmarshal(obj, &copyClone)
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
	obj := panicMarshal(copy)
	copyClone := ecs.RegisterTaskDefinitionInput{}
	panicUnmarshal(obj, &copyClone)
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

func copyTaskDef(api ecsiface.ECSAPI, taskdef string, imageMap map[string]string, envMaps map[string]map[string]string, secretMaps map[string]map[string]string, logopts map[string]map[string]map[string]string, logsecrets map[string]map[string]map[string]string, taskRole string) (string, error) {
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
	taskDefinitionArn := tdNew.TaskDefinition.TaskDefinitionArn
	return *taskDefinitionArn, nil
}

func alterService(api ecsiface.ECSAPI, cluster, service string, imageMap map[string]string, envMaps map[string]map[string]string, secretMaps map[string]map[string]string, logopts map[string]map[string]map[string]string, logsecrets map[string]map[string]map[string]string, taskRole string, desiredCount *int64, taskdef string) (ecs.Service, ecs.Service, error) {
	output, err := api.DescribeServices(&ecs.DescribeServicesInput{Cluster: aws.String(cluster), Services: []*string{aws.String(service)}})
	if err != nil {
		return ecs.Service{}, ecs.Service{}, err
	}
	copyTaskDefinitionAction := func(sourceTaskDefinition string) (string, error) {
		return copyTaskDef(api, sourceTaskDefinition, imageMap, envMaps, secretMaps, logopts, logsecrets, taskRole)
	}
	updateAction := func(newTaskDefinition *string, desiredCount *int64) (*ecs.UpdateServiceOutput, error) {
		updateServiceInput := &ecs.UpdateServiceInput{
			Cluster:            aws.String(cluster),
			Service:            aws.String(service),
			TaskDefinition:     newTaskDefinition,
			DesiredCount:       desiredCount,
			ForceNewDeployment: aws.Bool(true),
		}
		return api.UpdateService(updateServiceInput)
	}
	return findAndUpdateService(output, cluster, service, taskdef, desiredCount, copyTaskDefinitionAction, updateAction)
}

func findAndUpdateService(output *ecs.DescribeServicesOutput, cluster, service, taskDefinition string, desiredCount *int64, copyTdAction func(string) (string, error), updateSvcAction func(*string, *int64) (*ecs.UpdateServiceOutput, error)) (ecs.Service, ecs.Service, error) {
	if len(output.Services) == 0 {
		return ecs.Service{}, ecs.Service{}, ErrServiceNotFound
	}
	svc := output.Services[0]
	clusterArn := *svc.ClusterArn
	parsedClusterArn, err := arn.Parse(clusterArn)
	if err != nil {
		return ecs.Service{}, ecs.Service{}, err
	}
	return updateService(parsedClusterArn, svc, cluster, service, taskDefinition, desiredCount, copyTdAction, updateSvcAction)
}

func updateService(parsedClusterArn arn.ARN, svc *ecs.Service, cluster, service, td string, desiredCount *int64, copyTdAction func(string) (string, error), updateSvcAction func(*string, *int64) (*ecs.UpdateServiceOutput, error)) (ecs.Service, ecs.Service, error) {
	clusterNameFound := strings.TrimPrefix(parsedClusterArn.Resource, "cluster/")
	serviceNameFound := *svc.ServiceName
	if clusterNameFound == cluster && serviceNameFound == service {
		srcTaskDef := svc.TaskDefinition
		if td != "" {
			srcTaskDef = &td
		}
		newTd, err := copyTdAction(*srcTaskDef)
		if err != nil {
			return *svc, ecs.Service{}, err
		}
		if desiredCount == nil {
			desiredCount = svc.DesiredCount
		}
		updated, err := updateSvcAction(aws.String(newTd), desiredCount)
		if err != nil {
			return *svc, ecs.Service{}, err
		}
		return *svc, *updated.Service, nil
	}
	return ecs.Service{}, ecs.Service{}, ErrServiceNotFound
}

func mapStringStringAsJson(input map[string]string) string {
	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	_ = encoder.Encode(input)
	return strings.TrimSpace(buf.String())
}

func getTargetStates(targetGroupArn string, elbv2api elbv2iface.ELBV2API) (map[string]string, error) {
	describeLbOutput, err := elbv2api.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
		TargetGroupArn: aws.String(targetGroupArn),
	})
	if err != nil {
		return nil, err
	}
	targetStates := map[string]string{}
	for _, desc := range describeLbOutput.TargetHealthDescriptions {
		target := desc.Target
		health := desc.TargetHealth
		targetStates[*target.Id] = *health.State
	}
	return targetStates, nil
}

func validateDraining(ecsapi ecsiface.ECSAPI, elbv2api elbv2iface.ELBV2API, ecsService ecs.Service, bo backoff.BackOff) error {
	describeEcsOutput, err := ecsapi.DescribeServices(&ecs.DescribeServicesInput{Cluster: ecsService.ClusterArn, Services: []*string{ecsService.ServiceName}})

	if err != nil {
		return backoff.Permanent(err)
	}
	if len(describeEcsOutput.Services) == 0 {
		return backoff.Permanent(ErrServiceNotFound)
	}
	service := describeEcsOutput.Services[0]
	if len(service.LoadBalancers) == 0 {
		return ErrLoadBalancerNotConfigured
	}
	loadBalancer := service.LoadBalancers[0]
	targetGroupArn := loadBalancer.TargetGroupArn

	initialTargetIdState, err := getTargetStates(*targetGroupArn, elbv2api)
	if err != nil {
		return backoff.Permanent(err)
	}

	initTargetState := mapStringStringAsJson(initialTargetIdState)
	log.Printf("Initial target states: '%s'", initTargetState)

	operation := func() error {
		newTargetIdState, err := getTargetStates(*targetGroupArn, elbv2api)

		if err != nil {
			log.Print(err)
			return err
		}

		newTargetState := mapStringStringAsJson(newTargetIdState)
		log.Printf("Waiting for targets transitioning to draining state: '%s'", newTargetState)

		for targetId, initialTargetState := range initialTargetIdState {
			newTargetState := newTargetIdState[targetId]
			if initialTargetState != newTargetState && newTargetState == elbv2.TargetHealthStateEnumDraining {
				log.Printf("The target '%s' transitioned to draining state", targetId)
				return nil
			}
		}

		allInitialTargetsGone := true
		for targetId, _ := range initialTargetIdState {
			if _, found := newTargetIdState[targetId]; found {
				allInitialTargetsGone = false
			}
		}

		if allInitialTargetsGone {
			log.Printf("Either there are no initial targets or all targets are new or the service desired count was set to 0")
			return nil
		}

		return ErrWaitingForDrainingState
	}

	return backoff.Retry(operation, bo)
}

func validateDeployment(api ecsiface.ECSAPI, _ elbv2iface.ELBV2API, ecsService ecs.Service, bo backoff.BackOff) error {
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

func alterServiceValidateDeployment(ecsapi ecsiface.ECSAPI, elbv2api elbv2iface.ELBV2API, cluster, service string, imageMap map[string]string, envMaps map[string]map[string]string, secretMaps map[string]map[string]string, logopts map[string]map[string]map[string]string, logsecrets map[string]map[string]map[string]string, taskRole string, desiredCount *int64, taskdef string, bo backoff.BackOff, validateDeployment validateDeploymentFunc) (ecs.Service, error) {
	oldsvc, newsvc, err := alterService(ecsapi, cluster, service, imageMap, envMaps, secretMaps, logopts, logsecrets, taskRole, desiredCount, taskdef)
	if err != nil {
		return oldsvc, err
	}
	var prevErr error
	operation := func() error {
		err := validateDeployment(ecsapi, elbv2api, newsvc, bo)
		if err != prevErr && err != nil {
			prevErr = err
			log.Print(err)
		}
		return err
	}
	return oldsvc, backoff.Retry(operation, bo)
}

const (
	WaitUntilPrimaryRolled   = "primary-rolled"
	WaitUntilDrainingStarted = "draining-started"
)

var WaitUntilOptionList = []string{WaitUntilPrimaryRolled, WaitUntilDrainingStarted}

// ECSServiceUpdate encapsulates the attributes of an ECS service update
type ECSServiceUpdate struct {
	EcsApi           ecsiface.ECSAPI                         // ECS Api
	ElbApi           elbv2iface.ELBV2API                     // ELBV2 Api
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
	WaitUntil        *string                                 // Decide wether to wait until the service "started-draining" (only valid for services with Load Balancers attached) or until the deployment "primary-rolled" (default)
}

// Apply the ECS Service Update
func (e *ECSServiceUpdate) Apply() error {
	var useValidateDeploymentFunc validateDeploymentFunc = validateDeployment

	if e.WaitUntil != nil {
		switch *e.WaitUntil {
		case WaitUntilDrainingStarted:
			useValidateDeploymentFunc = validateDraining
			break
		case WaitUntilPrimaryRolled:
			useValidateDeploymentFunc = validateDeployment
			break
		default:
			return ErrInvalidWaitUntil
		}
	}
	return alterServiceOrValidatedRollBack(e.EcsApi, e.ElbApi, e.Cluster, e.Service, e.Image, e.Environment, e.Secrets, e.LogDriverOptions, e.LogDriverSecrets, e.TaskRole, e.DesiredCount, e.Taskdef, e.BackOff, useValidateDeploymentFunc)
}
