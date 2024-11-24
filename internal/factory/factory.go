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
	Run(ruleName, namespace string, replicas types.ScalingOperation) error
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

func readReplicas(ctx context.Context, sc *store.Persistence, persistence bool, defaultScalingObject *store.ScalingOperation) error {
	if !persistence {
		return ErrNotErrorDisabledPersitence
	}

	if err := sc.ScalingOperation.Get(ctx, defaultScalingObject); err != nil {
		return err
	}

	return nil
}

func writeReplicas(ctx context.Context, sc *store.Persistence, persistence bool, currentObjectReplicas int32, defaultScalingObject *store.ScalingOperation) error {
	if !persistence {
		return ErrNotErrorDisabledPersitence
	}

	operationTypeReplicas := defaultScalingObject.Replicas
	defaultScalingObject.Replicas = int(currentObjectReplicas)

	if err := sc.ScalingOperation.Update(ctx, defaultScalingObject); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if err := sc.ScalingOperation.Insert(ctx, defaultScalingObject); err != nil {
				return err
			}
		}
	}

	defaultScalingObject.Replicas = operationTypeReplicas
	return nil
}

func (sc *ScaleDeployment) Run(RuleNameDescription, objectNamespace string, operationTypeReplicas types.ScalingOperation) error {
	var deployments appsv1.DeploymentList
	if err := sc.client.Get(objectNamespace, &deployments); err != nil {
		return err
	}

	for _, deployment := range deployments.Items {
		currentObjectReplicas := *deployment.Spec.Replicas

		defaultScalingObjectValues := store.ScalingOperation{
			ResourceName:        deployment.Name,
			RuleNameDescription: RuleNameDescription,
			NamespaceName:       objectNamespace,
			ResourceType:        DEPLOYMENT,
			Replicas:            int(operationTypeReplicas),
		}

		if operationTypeReplicas == types.OperationDownscale {
			if err := writeReplicas(
				context.Background(),
				sc.storeClient,
				sc.persistence,
				currentObjectReplicas,
				&defaultScalingObjectValues,
			); err != nil {
				if !errors.Is(err, ErrNotErrorDisabledPersitence) {
					return err
				}
			}
		}

		if operationTypeReplicas == types.OperationUpscale {
			if err := readReplicas(
				context.Background(),
				sc.storeClient,
				sc.persistence,
				&defaultScalingObjectValues,
			); err != nil {
				if !errors.Is(err, ErrNotErrorDisabledPersitence) {
					sc.logger.Error(err, "database", "reading replicas error", err)
					return err
				}
			}
		}

		if err := sc.client.Patch(defaultScalingObjectValues.Replicas, &deployment); err != nil {
			sc.logger.Error(err, "client", "error patching deployment", err)
			return err
		}

		sc.logger.Info("client", "patching deployment", deployment.Name, "namespace", objectNamespace, "before", currentObjectReplicas, "after", defaultScalingObjectValues.Replicas, "object", defaultScalingObjectValues)
	}

	return nil
}

type ScaleStatefulSet struct {
	client *client.APIClient
	logger logr.Logger

	persistence bool
	storeClient *store.Persistence
}

func (sc *ScaleStatefulSet) Run(ruleNameDescription, objectNamespace string, operationTypeReplicas types.ScalingOperation) error {
	var statefulSets appsv1.StatefulSetList
	if err := sc.client.Get(objectNamespace, &statefulSets); err != nil {
		return err
	}

	for _, statefulSet := range statefulSets.Items {
		currentObjectReplicas := *statefulSet.Spec.Replicas

		defaultScalingObjectValues := store.ScalingOperation{
			RuleNameDescription: ruleNameDescription,
			ResourceName:        statefulSet.Name,
			NamespaceName:       objectNamespace,
			ResourceType:        STATEFULSET,
			Replicas:            int(operationTypeReplicas),
		}

		if operationTypeReplicas == types.OperationDownscale {
			if err := writeReplicas(context.Background(),
				sc.storeClient,
				sc.persistence,
				currentObjectReplicas,
				&defaultScalingObjectValues,
			); err != nil {
				if !errors.Is(err, ErrNotErrorDisabledPersitence) {
					return err
				}
			}
		}

		if operationTypeReplicas == types.OperationUpscale {
			if err := readReplicas(
				context.Background(),
				sc.storeClient,
				sc.persistence,
				&defaultScalingObjectValues,
			); err != nil {
				if !errors.Is(err, ErrNotErrorDisabledPersitence) {
					sc.logger.Error(err, "database", "reading replicas error", err)
					return err
				}
			}
		}

		if err := sc.client.Patch(defaultScalingObjectValues.Replicas, &statefulSet); err != nil {
			sc.logger.Error(err, "client", "error patching deployment", err)
			return err
		}

		sc.logger.Info("client", "patching statefulSet", statefulSet.Name, "namespace", objectNamespace, "before", currentObjectReplicas, "after", defaultScalingObjectValues.Replicas, "object", defaultScalingObjectValues)
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
