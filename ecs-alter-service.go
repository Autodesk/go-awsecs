package awsecs

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go/service/elbv2/elbv2iface"
	"github.com/cenkalti/backoff"
	"log"
)

var (
	// ErrNothingToRollback nothing to rollback
	ErrNothingToRollback = errors.New("nothing to rollback")
	// ErrPermanentNothingToRollback permanent nothing to rollback
	ErrPermanentNothingToRollback = backoff.Permanent(ErrNothingToRollback)
	// ErrSuccessfulRollback successful rollback
	ErrSuccessfulRollback = errors.New("successful rollback")
	// ErrFailedRollback failed rollback
	ErrFailedRollback = errors.New("failed rollback")
)

type validateDeploymentFunc func(ecsiface.ECSAPI, elbv2iface.ELBV2API, ecs.Service, backoff.BackOff) error

func alterServiceOrValidatedRollBack(ecsapi ecsiface.ECSAPI, elbv2api elbv2iface.ELBV2API, cluster, service string, imageMap map[string]string, envMaps map[string]map[string]string, secretMaps map[string]map[string]string, logopts map[string]map[string]map[string]string, logsecrets map[string]map[string]map[string]string, taskRole string, desiredCount *int64, taskdef string, bo backoff.BackOff, validateDeployment validateDeploymentFunc) error {
	oldsvc, alterSvcErr := alterServiceValidateDeployment(ecsapi, elbv2api, cluster, service, imageMap, envMaps, secretMaps, logopts, logsecrets, taskRole, desiredCount, taskdef, bo, validateDeployment)
	if alterSvcErr != nil {
		operation := func() error {
			if oldsvc.ServiceName == nil {
				return ErrPermanentNothingToRollback
			}
			log.Printf("attempt rollback %v", alterSvcErr)
			rollback, err := ecsapi.UpdateService(&ecs.UpdateServiceInput{Cluster: oldsvc.ClusterArn, Service: oldsvc.ServiceName, TaskDefinition: oldsvc.TaskDefinition, DesiredCount: oldsvc.DesiredCount, ForceNewDeployment: aws.Bool(true)})
			if err != nil {
				return err
			}
			var prevErr error
			operation := func() error {
				err := validateDeployment(ecsapi, elbv2api, *rollback.Service, bo)
				if err != prevErr && err != nil {
					prevErr = err
					log.Print(err)
				}
				return err
			}
			return backoff.Retry(operation, bo)
		}
		if err := backoff.Retry(operation, bo); err != nil {
			if err == ErrNothingToRollback {
				return alterSvcErr
			}
			return ErrFailedRollback
		}
		return ErrSuccessfulRollback
	}
	return alterSvcErr
}
