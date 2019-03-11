package awsecs

import (
	"errors"
	"github.com/aws/aws-sdk-go/service/ecs"
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

func alterServiceOrValidatedRollBack(api ecs.ECS, cluster, service string, imageMap map[string]string, envMaps map[string]map[string]string, secretMaps map[string]map[string]string, desiredCount *int64, bo backoff.BackOff) error {
	oldsvc, alterSvcErr := alterServiceValidateDeployment(api, cluster, service, imageMap, envMaps, secretMaps, desiredCount, bo)
	if alterSvcErr != nil {
		operation := func() error {
			if oldsvc.ServiceName == nil {
				return ErrPermanentNothingToRollback
			}
			log.Printf("attempt rollback %v", alterSvcErr)
			rollback, err := api.UpdateService(&ecs.UpdateServiceInput{Cluster: oldsvc.ClusterArn, Service: oldsvc.ServiceName, TaskDefinition: oldsvc.TaskDefinition, DesiredCount: oldsvc.DesiredCount})
			if err != nil {
				return err
			}
			operation := func() error {
				err := validateDeployment(api, *rollback.Service)
				if err != nil {
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
