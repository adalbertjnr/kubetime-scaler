package client

import (
	"context"

	downscalergov1alpha1 "github.com/adalbertjnr/downscalerk8s/api/v1alpha1"
	"github.com/adalbertjnr/downscalerk8s/internal/pkgerrors"
	appsv1 "k8s.io/api/apps/v1"
	v2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type APIClient struct {
	client.Client

	ctx context.Context
}

func NewAPIClient(c client.Client) *APIClient {
	return &APIClient{
		ctx:    context.Background(),
		Client: c,
	}
}

func (c *APIClient) GetDeployment(name, namespace string) (*appsv1.Deployment, error) {
	var deployment appsv1.Deployment

	clientObjectKey := client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	if err := c.Client.Get(c.ctx, clientObjectKey, &deployment); err != nil {
		return nil, err
	}

	return &deployment, nil
}

func (c *APIClient) GetNamespaces() (*v1.NamespaceList, error) {
	var namespaces v1.NamespaceList

	if err := c.Client.List(c.ctx, &namespaces); err != nil {
		return nil, err
	}

	return &namespaces, nil
}

func (c *APIClient) Patch(replicas int, object any) error {
	patchOpts := client.Merge
	replicaCount := int32(replicas)

	switch value := object.(type) {
	case *appsv1.Deployment:
		value.Spec.Replicas = &replicaCount
		return c.Client.Patch(c.ctx, value, patchOpts)
	case *appsv1.StatefulSet:
		value.Spec.Replicas = &replicaCount
		return c.Client.Patch(c.ctx, value, patchOpts)
	case *v2.HorizontalPodAutoscaler:
		value.Spec.MinReplicas = &replicaCount
		return c.Client.Patch(c.ctx, value, patchOpts)
	default:
		return pkgerrors.ErrPatchingTypeNotFound
	}
}

func (c *APIClient) Get(namespace string, resource any) error {
	listOpts := &client.ListOptions{Namespace: namespace}

	switch value := resource.(type) {
	case *appsv1.DeploymentList:
		return c.Client.List(c.ctx, value, listOpts)
	case *appsv1.StatefulSetList:
		return c.Client.List(c.ctx, value, listOpts)
	case *v2.HorizontalPodAutoscalerList:
		return c.Client.List(c.ctx, value, listOpts)
	default:
		return pkgerrors.ErrListTypeNotFound
	}
}

func (c *APIClient) GetDeployments(namespace string) (*appsv1.DeploymentList, error) {
	var deploymentList appsv1.DeploymentList

	listOpts := &client.ListOptions{Namespace: namespace}

	if err := c.Client.List(c.ctx, &deploymentList, listOpts); err != nil {
		return nil, err
	}

	return &deploymentList, nil
}

func (c *APIClient) GetStatefulSetList(namespace string) (*appsv1.StatefulSetList, error) {
	var statefulSetList appsv1.StatefulSetList

	listOpts := &client.ListOptions{Namespace: namespace}
	if err := c.Client.List(c.ctx, &statefulSetList, listOpts); err != nil {
		return nil, err
	}

	return &statefulSetList, nil
}

func (c *APIClient) GetHPAList(namespace string) (*v2.HorizontalPodAutoscalerList, error) {
	var hpaList v2.HorizontalPodAutoscalerList

	listOpts := &client.ListOptions{Namespace: namespace}
	if err := c.Client.List(c.ctx, &hpaList, listOpts); err != nil {
		return nil, err
	}

	return &hpaList, nil
}

func (c *APIClient) GetDownscaler() (downscaler downscalergov1alpha1.Downscaler, err error) {
	if err := c.Client.Get(context.Background(), types.NamespacedName{
		Name:      "downscaler",
		Namespace: "downscaler",
	}, &downscaler); err != nil {
		return downscaler, err
	}

	return downscaler, nil
}
