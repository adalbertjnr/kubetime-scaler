package client

import (
	"context"

	downscalergov1alpha1 "github.com/adalbertjnr/downscaler-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
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

func (c *APIClient) GetDeployments(namespace string) (*appsv1.DeploymentList, error) {
	var deploymentList appsv1.DeploymentList

	listOpts := &client.ListOptions{Namespace: namespace}

	if err := c.Client.List(c.ctx, &deploymentList, listOpts); err != nil {
		return nil, err
	}

	return &deploymentList, nil
}

func (c *APIClient) PatchDeployment(replicas int, deployment *appsv1.Deployment) error {

	patchOpts := client.Merge

	r := int32(replicas)
	deployment.Spec.Replicas = &r

	return c.Client.Patch(c.ctx, deployment, patchOpts)
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
