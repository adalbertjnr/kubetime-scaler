package factory

import (
	"log/slog"

	"github.com/adalbertjnr/downscalerk8s/internal/client"
	"github.com/adalbertjnr/downscalerk8s/internal/store"
	appsv1 "k8s.io/api/apps/v1"
	v2 "k8s.io/api/autoscaling/v2"
)

const (
	DEPLOYMENT  = "deployments"
	STATEFULSET = "statefulset"
	HPA         = "hpa"
)

type ResourceScaler interface {
	Run(namespace string, replicas int) error
}

type ScaleDeployment struct {
	client      *client.APIClient
	persistence *store.Persistence
}

func (sc *ScaleDeployment) Run(namespace string, replicas int) error {
	var deployments appsv1.DeploymentList
	if err := sc.client.Get(namespace, &deployments); err != nil {
		return err
	}

	for _, deployment := range deployments.Items {
		before := *deployment.Spec.Replicas

		if err := sc.client.Patch(replicas, &deployment); err != nil {
			slog.Error("client", "error patching deployment", err)
			return err
		}

		slog.Info("client", "patching deployment", deployment.Name, "namespace", namespace, "before", before, "after", replicas)
	}

	return nil
}

type ScaleHPA struct {
	client      *client.APIClient
	persistence *store.Persistence
}

func (sc *ScaleHPA) Run(namespace string, replicas int) error {
	var hpaList v2.HorizontalPodAutoscalerList
	if err := sc.client.Get(namespace, &hpaList); err != nil {
		return err
	}

	for _, hpa := range hpaList.Items {
		before := *hpa.Spec.MinReplicas

		if err := sc.client.Patch(replicas, &hpa); err != nil {
			slog.Error("client", "error patching deployment", err)
			return err
		}

		slog.Info("client", "patching hpa", hpa.Name, "namespace", namespace, "before", before, "after", replicas)
	}

	return nil
}

type ScaleStatefulSet struct {
	client      *client.APIClient
	persistence *store.Persistence
}

func (sc *ScaleStatefulSet) Run(namespace string, replicas int) error {
	var statefulSets appsv1.StatefulSetList
	if err := sc.client.Get(namespace, &statefulSets); err != nil {
		return err
	}

	for _, statefulSet := range statefulSets.Items {
		before := *statefulSet.Spec.Replicas

		if err := sc.client.Patch(replicas, &statefulSet); err != nil {
			slog.Error("client", "error patching deployment", err)
			return err
		}

		slog.Info("client", "patching statefulSet", statefulSet.Name, "namespace", namespace, "before", before, "after", replicas)
	}

	return nil
}

type FactoryScaler map[string]ResourceScaler

func NewScalerFactory(persistence *store.Persistence, client *client.APIClient) *FactoryScaler {
	return &FactoryScaler{
		DEPLOYMENT: &ScaleDeployment{
			persistence: persistence,
			client:      client,
		},

		STATEFULSET: &ScaleStatefulSet{
			persistence: persistence,
			client:      client,
		},

		HPA: &ScaleHPA{
			persistence: persistence,
			client:      client,
		},
	}
}
