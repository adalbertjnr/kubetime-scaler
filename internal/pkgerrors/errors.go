package pkgerrors

import "errors"

var (
	ErrMalformedTimeZone         = errors.New("invalid time zone format: expected 'Region/City'")
	ErrNilInclude                = errors.New("include rules must be provided")
	ErrRulesNotProvided          = errors.New("rules configuration block is missing")
	ErrTimeRulesBlockNotProvided = errors.New("the user must provide the timeRules")
	ErrEmptyNamespaces           = errors.New("namespace list cannot be empty")
	ErrMalforedDownscaleTime     = errors.New("provided downscaleTime is malformed")
	ErrMalforedUpscaleTime       = errors.New("provided upscaleTime is malformed")
	ErrPatchingTypeNotFound      = errors.New("the provided type for patching do not exists")
	ErrListTypeNotFound          = errors.New("the provided resource for listing do not exists")
)
