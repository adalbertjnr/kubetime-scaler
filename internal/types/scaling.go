package types

type ScalingOperation int

const (
	OperationDownscale ScalingOperation = iota
	OperationUpscale
)
