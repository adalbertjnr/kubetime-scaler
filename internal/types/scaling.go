package types

type ScalingOperation int

const (
	OperationDownscale ScalingOperation = iota
	OperationUpscale
)

type ResourceType string

const (
	DeploymentObjectResource  ResourceType = "deployments"
	StatefulSetObjectResource ResourceType = "statefulset"
)

func (r ResourceType) String() string {
	return string(r)
}
