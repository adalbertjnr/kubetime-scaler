package scheduler

import (
	"log/slog"
	"strings"

	"github.com/adalbertjnr/downscaler-operator/api/v1alpha1"
	"github.com/adalbertjnr/downscaler-operator/internal/pkgerrors"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	spec                  = "spec"
	schedule              = "schedule"
	timeZone              = "timeZone"
	namespaces            = "namespaces"
	namespacesRules       = "namespacesRules"
	include               = "include"
	WithRulesByNamespaces = "withRulesByNamespaces"
	rules                 = "rules"
	upscaleTime           = "upscaleTime"
	downscaleTime         = "downscaleTime"
)

func (s *Downscaler) Validate() bool {
	valid := true

	spec := s.app.Spec

	var validationErrors []error
	processScheduleFields(
		&spec.Schedule,
		&validationErrors,
	)

	processIncludeFields(
		spec.NamespacesRules.Include,
		&validationErrors,
	)

	if len(validationErrors) > 0 {
		for _, err := range validationErrors {
			slog.Error("validation failed", "err", err)
		}
		return !valid
	}

	return valid
}

func processScheduleFields(scheduleField *v1alpha1.Schedule, validationErrors *[]error) {
	if scheduleField == nil {
		*validationErrors = append(*validationErrors, field.Invalid(
			field.NewPath(spec).Child(schedule),
			scheduleField,
			pkgerrors.ErrNilInclude.Error(),
		))
		return
	}

	if scheduleField.TimeZone == "" || len(strings.Split(scheduleField.TimeZone, "/")) == 1 {
		*validationErrors = append(*validationErrors, field.Invalid(
			field.NewPath(spec).Child(schedule).Child(timeZone),
			scheduleField.TimeZone,
			pkgerrors.ErrMalformedTimeZone.Error(),
		))
	}
}

func processIncludeFields(includeField *v1alpha1.Include, validationErrors *[]error) {
	childBase := field.NewPath(spec).Child(namespacesRules).Child(include)

	if includeField == nil {
		*validationErrors = append(*validationErrors, field.Invalid(
			childBase,
			includeField,
			pkgerrors.ErrNilInclude.Error(),
		))
		return
	}

	if includeField.WithRulesByNamespaces == nil {
		*validationErrors = append(*validationErrors, field.Invalid(
			childBase.Child(WithRulesByNamespaces),
			includeField.WithRulesByNamespaces,
			pkgerrors.ErrNilWithRulesByNamespaces.Error(),
		))
		return
	}

	if includeField.WithRulesByNamespaces.Rules == nil {
		*validationErrors = append(*validationErrors, field.Invalid(
			childBase.Child(WithRulesByNamespaces).Child(rules),
			includeField.WithRulesByNamespaces.Rules,
			pkgerrors.ErrRulesNotProvided.Error(),
		))
		return
	}

	for index, rule := range includeField.WithRulesByNamespaces.Rules {
		childRule := childBase.Child(WithRulesByNamespaces).Index(index)

		if len(rule.Namespaces) == 0 {
			*validationErrors = append(*validationErrors, field.Invalid(
				childRule.Child(namespaces),
				rule.Namespaces,
				pkgerrors.ErrEmptyNamespaces.Error(),
			))
		}

		if len(strings.Split(rule.UpscaleTime, ":")) == 1 {
			*validationErrors = append(*validationErrors, field.Invalid(
				childRule.Child(namespaces).Child(upscaleTime),
				rule.UpscaleTime,
				pkgerrors.ErrMalforedUpscaleTime.Error(),
			))
		}

		if len(strings.Split(rule.DownscaleTime, ":")) == 1 {
			*validationErrors = append(*validationErrors, field.Invalid(
				childRule.Child(namespaces).Child(downscaleTime),
				rule.DownscaleTime,
				pkgerrors.ErrMalforedDownscaleTime.Error(),
			))
		}

	}
}
