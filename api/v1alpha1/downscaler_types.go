/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DownscalerSpec defines the desired state of Downscaler
type DownscalerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Config            Config            `json:"config"`
	Schedule          Schedule          `json:"schedule"`
	DownscalerOptions DownscalerOptions `json:"downscalerOptions"`
}

type Config struct {
	CronLoggerInterval int `json:"cronLoggerInterval,omitempty"`
}

type Schedule struct {
	TimeZone   string `json:"timeZone,omitempty"`
	Recurrence string `json:"recurrence,omitempty"`
}

type TimeRules struct {
	Rules []Rules `json:"rules"`
}

type DownscalerOptions struct {
	TimeRules *TimeRules `json:"timeRules"`
}

type Namespace string

func (ns Namespace) Ignored(e map[string]struct{}, downscalerNamespace string) bool {
	if _, found := e[downscalerNamespace]; found {
		return true
	}
	return false
}

func (ns Namespace) String() string {
	return string(ns)
}

type Rules struct {
	Namespaces    []Namespace `json:"namespaces,omitempty"`
	UpscaleTime   string      `json:"upscaleTime,omitempty"`
	DownscaleTime string      `json:"downscaleTime,omitempty"`
}

// DownscalerStatus defines the observed state of Downscaler
type DownscalerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Downscaler is the Schema for the downscalers API
type Downscaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DownscalerSpec   `json:"spec,omitempty"`
	Status DownscalerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DownscalerList contains a list of Downscaler
type DownscalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Downscaler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Downscaler{}, &DownscalerList{})
}
