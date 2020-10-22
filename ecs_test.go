package awsecs

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ecs"
	"reflect"
	"testing"
)

func TestUpdateService(t *testing.T) {
	type args struct {
		parsedClusterArn arn.ARN
		svc              *ecs.Service
		cluster          string
		service          string
		td               string
		desiredCount     *int64
		copyTdAction     func(string) (string, error)
		updateSvcAction  func(*string, *int64) (*ecs.UpdateServiceOutput, error)
	}
	tests := []struct {
		name         string
		wantErr      bool
		beforeUpdate ecs.Service
		afterUpdate  ecs.Service
		args         args
	}{
		{
			name:    "On copy error I want error",
			wantErr: true,
			beforeUpdate: ecs.Service{
				ServiceName: aws.String("my-service"),
			},
			afterUpdate: ecs.Service{},
			args: args{
				parsedClusterArn: arn.ARN{
					Resource: "cluster/my-cluster",
				},
				svc: &ecs.Service{
					ServiceName: aws.String("my-service"),
				},
				cluster:      "my-cluster",
				service:      "my-service",
				td:           "task:1",
				desiredCount: aws.Int64(1),
				copyTdAction: func(string) (string, error) {
					return "", errors.New("failed to copy")
				},
				updateSvcAction: nil,
			},
		},
		{
			name:    "On update error I want error",
			wantErr: true,
			beforeUpdate: ecs.Service{
				ServiceName: aws.String("my-service"),
			},
			afterUpdate: ecs.Service{},
			args: args{
				parsedClusterArn: arn.ARN{
					Resource: "cluster/my-cluster",
				},
				svc: &ecs.Service{
					ServiceName: aws.String("my-service"),
				},
				cluster:      "my-cluster",
				service:      "my-service",
				td:           "task:1",
				desiredCount: aws.Int64(1),
				copyTdAction: func(string) (string, error) {
					return "task:2", nil
				},
				updateSvcAction: func(*string, *int64) (*ecs.UpdateServiceOutput, error) {
					return nil, errors.New("failed to update")
				},
			},
		},
		{
			name:         "On non matching cluster I want error",
			wantErr:      true,
			beforeUpdate: ecs.Service{},
			afterUpdate:  ecs.Service{},
			args: args{
				parsedClusterArn: arn.ARN{
					Resource: "cluster/my-cluster",
				},
				svc: &ecs.Service{
					ServiceName: aws.String("my-service"),
				},
				cluster: "my-other-cluster",
				service: "my-service",
			},
		},
		{
			name:         "On non matching service I want error",
			wantErr:      true,
			beforeUpdate: ecs.Service{},
			afterUpdate:  ecs.Service{},
			args: args{
				parsedClusterArn: arn.ARN{
					Resource: "cluster/my-cluster",
				},
				svc: &ecs.Service{
					ServiceName: aws.String("my-service"),
				},
				cluster: "my-cluster",
				service: "my-other-service",
			},
		},
		{
			name:    "Check before and after update",
			wantErr: false,
			beforeUpdate: ecs.Service{
				ServiceName: aws.String("my-service"),
			},
			afterUpdate: ecs.Service{
				TaskDefinition: aws.String("task:2"),
			},
			args: args{
				parsedClusterArn: arn.ARN{
					Resource: "cluster/my-cluster",
				},
				svc: &ecs.Service{
					ServiceName: aws.String("my-service"),
				},
				cluster:      "my-cluster",
				service:      "my-service",
				td:           "task:1",
				desiredCount: nil,
				copyTdAction: func(s string) (string, error) {
					return "task:2", nil
				},
				updateSvcAction: func(s *string, i *int64) (*ecs.UpdateServiceOutput, error) {
					return &ecs.UpdateServiceOutput{
						Service: &ecs.Service{
							TaskDefinition: aws.String("task:2"),
						},
					}, nil
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := updateService(tt.args.parsedClusterArn, tt.args.svc, tt.args.cluster, tt.args.service, tt.args.td, tt.args.desiredCount, tt.args.copyTdAction, tt.args.updateSvcAction)
			if (err != nil) != tt.wantErr {
				t.Errorf("updateService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.beforeUpdate) {
				t.Errorf("updateService() got = %v, want %v", got, tt.beforeUpdate)
			}
			if !reflect.DeepEqual(got1, tt.afterUpdate) {
				t.Errorf("updateService() got1 = %v, want %v", got1, tt.afterUpdate)
			}
		})
	}
}
