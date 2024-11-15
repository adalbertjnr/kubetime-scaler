package pkgerrors

import "errors"

var (
	ErrMalformedTimeZone = errors.New("invalid time zone format: expected 'Region/City'")

	ErrNilInclude = errors.New("include rules must be provided")

	ErrNilWithRulesByNamespaces = errors.New("withRulesByNamespaces block is required")

	ErrRulesNotProvided = errors.New("rules configuration block is missing")

	ErrEmptyNamespaces = errors.New("namespace list cannot be empty")

	ErrMalforedDownscaleTime = errors.New("provided downscaleTime is malformed")

	ErrMalforedUpscaleTime = errors.New("provided upscaleTime is malformed")
)
