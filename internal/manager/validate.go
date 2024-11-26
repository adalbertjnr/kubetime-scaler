package manager

import (
	"log/slog"
	"strings"

	"github.com/adalbertjnr/downscalerk8s/api/v1alpha1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	Spec = "spec"

	Schedule = "schedule"
	TimeZone = "timeZone"

	DownscalerOptions = "downscalerOptions"
	TimeRules         = "timeRules"
	Rules             = "rules"
	Namespaces        = "namespaces"
	UpscaleTime       = "upscaleTime"
	DownscaleTime     = "downscaleTime"
)

func (s *Downscaler) Validate() bool {
	valid := true

	var validationErrors []error

	processScheduleFields(&s.app.Spec.Schedule, &validationErrors)
	processDownscalerOptions(&s.app.Spec.DownscalerOptions, &validationErrors)

	if len(validationErrors) > 0 {
		for _, err := range validationErrors {
			slog.Error("validation failed", "err", err)
		}
		return !valid
	}

	return valid
}

func processScheduleFields(schedule *v1alpha1.Schedule, validationErrors *[]error) {
	if schedule == nil {
		err := field.Invalid(field.NewPath(Spec).Child(Schedule), schedule, "")
		*validationErrors = append(*validationErrors, err)
		return
	}
	if schedule.TimeZone == "" || len(strings.Split(schedule.TimeZone, "/")) == 1 {
		err := field.Invalid(field.NewPath(Spec).Child(Schedule).Child(TimeZone), schedule.TimeZone, "")
		*validationErrors = append(*validationErrors, err)
	}
}

func processDownscalerOptions(options *v1alpha1.DownscalerOptions, validationErrors *[]error) {
	childBase := field.NewPath(Spec).Child(DownscalerOptions)

	if options == nil {
		err := field.Invalid(childBase, options, "")
		*validationErrors = append(*validationErrors, err)
		return
	}

	if options.TimeRules == nil {
		err := field.Invalid(childBase.Child(TimeRules), options.TimeRules, "")
		*validationErrors = append(*validationErrors, err)
		return
	}

	if options.TimeRules.Rules == nil {
		err := field.Invalid(childBase.Child(TimeRules).Child(Rules), options.TimeRules.Rules, "")
		*validationErrors = append(*validationErrors, err)
		return
	}

	for index, rule := range options.TimeRules.Rules {
		childRule := childBase.Child(TimeRules).Index(index)

		if len(rule.Namespaces) == 0 {
			err := field.Invalid(childRule.Child(Namespaces), rule.Namespaces, "")
			*validationErrors = append(*validationErrors, err)
		}

		if len(strings.Split(rule.UpscaleTime, ":")) == 1 {
			err := field.Invalid(childRule.Child(Namespaces).Child(UpscaleTime), rule.UpscaleTime, "")
			*validationErrors = append(*validationErrors, err)
		}

		if len(strings.Split(rule.DownscaleTime, ":")) == 1 {
			err := field.Invalid(childRule.Child(Namespaces).Child(DownscaleTime), rule.DownscaleTime, "")
			*validationErrors = append(*validationErrors, err)
		}

	}
}
