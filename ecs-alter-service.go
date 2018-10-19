package awsecs

import (
	"errors"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/cenkalti/backoff"
	"log"
)

var (
	// ErrNothingToRollback nothing to rollback
	ErrNothingToRollback = backoff.Permanent(errors.New("nothing to rollback"))
	// ErrSuccessfulRollback successful rollback
	ErrSuccessfulRollback = errors.New("successful rollback")
	// ErrFailedRollback failed rollback
	ErrFailedRollback = errors.New("failed rollback")
)

func alterServiceOrValidatedRollBack(api ecs.ECS, cluster, service string, imageMap map[string]string, envMaps map[string]map[string]string, desiredCount *int64, bo backoff.BackOff) error {
	oldsvc, err := alterServiceValidateDeployment(api, cluster, service, imageMap, envMaps, desiredCount, bo)
	if err != nil {
		operation := func() error {
			if oldsvc.ServiceName == nil {
				log.Print(ErrNothingToRollback)
				return ErrNothingToRollback
			}
			log.Print("attempt rollback")
			rollback, err := api.UpdateService(&ecs.UpdateServiceInput{Cluster: oldsvc.ClusterArn, Service: oldsvc.ServiceName, TaskDefinition: oldsvc.TaskDefinition, DesiredCount: oldsvc.DesiredCount})
			if err != nil {
				log.Print(err)
				return err
			}
			operation := func() error {
				err = validateDeployment(api, *rollback.Service)
				if err != nil {
					log.Print(err)
				}
				return err
			}
			return backoff.Retry(operation, bo)
		}
		if backoff.Retry(operation, bo) != nil {
			return ErrFailedRollback
		}
		return ErrSuccessfulRollback
	}
	return err
}
