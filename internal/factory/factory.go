package factory

import (
	"context"
	"database/sql"
	"errors"

	"github.com/adalbertjnr/downscalerk8s/internal/client"
	"github.com/adalbertjnr/downscalerk8s/internal/store"
	"github.com/adalbertjnr/downscalerk8s/internal/types"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
)

const (
	DEPLOYMENT  = "deployments"
	STATEFULSET = "statefulset"
)

type ResourceScaler interface {
	Run(namespace string, replicas types.ScalingOperation) error
}

type ScaleDeployment struct {
	client *client.APIClient
	logger logr.Logger

	persistence bool
	storeClient *store.Persistence
}

var (
	ErrNotErrorDisabledPersitence = errors.New("persistence is disabled")
	ErrNotErrorOperationUpscale   = errors.New("upscale operation. no need to write the replicas in the database")
	ErrNotErrorOperationDownscale = errors.New("downscale operation. no need to read the replicas in the database")
)

func readReplicas(ctx context.Context, logger logr.Logger, sc *store.Persistence, persistence bool, namespace string, resourceName string, operationTypeReplicas types.ScalingOperation) (int, error) {
	if !persistence {
		return int(operationTypeReplicas), ErrNotErrorDisabledPersitence
	}

	if operationTypeReplicas == types.OperationDownscale {
		return int(operationTypeReplicas), ErrNotErrorOperationDownscale
	}

	scalingObjectQuery := store.ScalingOperation{
		NamespaceName: namespace,
		ResourceName:  resourceName,
	}

	if err := sc.ScalingOperation.Get(ctx, &scalingObjectQuery); err != nil {
		return int(operationTypeReplicas), err
	}

	logger.Info("database", "reading object", scalingObjectQuery)
	return scalingObjectQuery.Replicas, nil
}

func writeReplicas(ctx context.Context, sc *store.Persistence, persistence bool, namespace string, resourceName string, before int32, operationType types.ScalingOperation, resourceType string) error {
	if !persistence {
		return ErrNotErrorDisabledPersitence
	}

	if operationType == types.OperationUpscale {
		return ErrNotErrorOperationUpscale
	}

	scalingObject := store.ScalingOperation{
		NamespaceName: namespace,
		ResourceType:  resourceType,
		Replicas:      int(before),
		ResourceName:  resourceName,
	}

	if err := sc.ScalingOperation.Update(ctx, &scalingObject); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if err := sc.ScalingOperation.Insert(ctx, &scalingObject); err != nil {
				return err
			}
		}
	}

	return nil
}

func (sc *ScaleDeployment) Run(namespace string, operationTypeReplicas types.ScalingOperation) error {
	var deployments appsv1.DeploymentList
	if err := sc.client.Get(namespace, &deployments); err != nil {
		return err
	}

	for _, deployment := range deployments.Items {
		currentObjectReplicas := *deployment.Spec.Replicas

		replicasResponse, err := readReplicas(
			context.Background(),
			sc.logger,
			sc.storeClient,
			sc.persistence,
			namespace,
			deployment.Name,
			operationTypeReplicas,
		)

		if !errors.Is(err, ErrNotErrorDisabledPersitence) || !errors.Is(err, ErrNotErrorOperationDownscale) {
			sc.logger.Error(err, "database", "reading replicas error", err)
			return err
		}

		if err := sc.client.Patch(replicasResponse, &deployment); err != nil {
			sc.logger.Error(err, "client", "error patching deployment", err)
			return err
		}

		if err := writeReplicas(
			context.Background(),
			sc.storeClient,
			sc.persistence,
			namespace,
			deployment.Name,
			currentObjectReplicas,
			operationTypeReplicas,
			DEPLOYMENT,
		); err != nil {
			if !errors.Is(err, ErrNotErrorDisabledPersitence) || !errors.Is(err, ErrNotErrorOperationUpscale) {
				return err
			}
		}

		sc.logger.Info("client", "patching deployment", deployment.Name, "namespace", namespace, "before", currentObjectReplicas, "after", operationTypeReplicas)
	}

	return nil
}

type ScaleStatefulSet struct {
	client *client.APIClient
	logger logr.Logger

	persistence bool
	storeClient *store.Persistence
}

func (sc *ScaleStatefulSet) Run(namespace string, operationTypeReplicas types.ScalingOperation) error {
	var statefulSets appsv1.StatefulSetList
	if err := sc.client.Get(namespace, &statefulSets); err != nil {
		return err
	}

	for _, statefulSet := range statefulSets.Items {
		currentObjectReplicas := *statefulSet.Spec.Replicas

		replicasResponse, err := readReplicas(
			context.Background(),
			sc.logger,
			sc.storeClient,
			sc.persistence,
			namespace,
			statefulSet.Name,
			operationTypeReplicas,
		)

		if !errors.Is(err, ErrNotErrorDisabledPersitence) || !errors.Is(err, ErrNotErrorOperationUpscale) {
			sc.logger.Error(err, "database", "reading replicas error", err)
			return err
		}

		if err := sc.client.Patch(replicasResponse, &statefulSet); err != nil {
			sc.logger.Error(err, "client", "error patching deployment", err)
			return err
		}

		if err := writeReplicas(
			context.Background(),
			sc.storeClient,
			sc.persistence,
			namespace,
			statefulSet.Name,
			currentObjectReplicas,
			operationTypeReplicas,
			STATEFULSET,
		); err != nil {
			if !errors.Is(err, ErrNotErrorDisabledPersitence) || !errors.Is(err, ErrNotErrorOperationUpscale) {
				return err
			}
		}

		sc.logger.Info("client", "patching statefulSet", statefulSet.Name, "namespace", namespace, "before", currentObjectReplicas, "after", operationTypeReplicas)
	}

	return nil
}

type FactoryScaler map[string]ResourceScaler

func NewScalerFactory(client *client.APIClient, store *store.Persistence, logger logr.Logger) *FactoryScaler {
	persistence := store != nil
	return &FactoryScaler{
		DEPLOYMENT: &ScaleDeployment{
			client:      client,
			logger:      logger,
			storeClient: store,
			persistence: persistence,
		},

		STATEFULSET: &ScaleStatefulSet{
			client:      client,
			logger:      logger,
			storeClient: store,
			persistence: persistence,
		},
	}
}
