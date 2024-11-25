package store

import "context"

type ScalingOperationStorer interface {
	Bootstrap(context.Context) error
	Get(context.Context, *ScalingOperation) error
	Update(context.Context, *ScalingOperation) error
	Insert(context.Context, *ScalingOperation) error
}

type ScalingOperation struct {
	ID                  int    `json:"id"`
	NamespaceName       string `json:"namespace_name"`
	RuleNameDescription string `json:"rule_name_description"`
	ResourceName        string `json:"resource_name"`
	ResourceType        string `json:"resource_type"`
	Replicas            int    `json:"replicas"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
}
