// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	commonv1alpha1 "github.com/cloudptio/logstash-operator/pkg/apis/common/v1alpha1"
)

const LogstashContainerName = "logstash"

// LogstashSpec defines the desired state of Logstash
type LogstashSpec struct {
	// Version represents the version of Logstash
	Version string `json:"version,omitempty"`

	// Image represents the docker image that will be used.
	Image string `json:"image,omitempty"`

	// NodeCount defines how many nodes the Logstash deployment must have.
	NodeCount int32 `json:"nodeCount,omitempty"`

	// ElasticsearchRef references an Elasticsearch resource in the Kubernetes cluster.
	// If the namespace is not specified, the current resource namespace will be used.
	ElasticsearchRef commonv1alpha1.ObjectSelector `json:"elasticsearchRef,omitempty"`

	// Config represents Logstash configuration.
	Config *commonv1alpha1.Config `json:"config,omitempty"`

	// HTTP contains settings for HTTP.
	HTTP commonv1alpha1.HTTPConfig `json:"http,omitempty"`

	// PodTemplate can be used to propagate configuration to Logstash pods.
	// This allows specifying custom annotations, labels, environment variables,
	// affinity, resources, etc. for the pods created from this spec.
	// +kubebuilder:validation:Optional
	PodTemplate corev1.PodTemplateSpec `json:"podTemplate,omitempty"`

	// SecureSettings references secrets containing secure settings, to be injected
	// into Logstash keystore on each node.
	// Each individual key/value entry in the referenced secrets is considered as an
	// individual secure setting to be injected.
	// You can use the `entries` and `key` fields to consider only a subset of the secret
	// entries and the `path` field to change the target path of a secret entry key.
	// The secret must exist in the same namespace as the Logstash resource.
	SecureSettings []commonv1alpha1.SecretSource `json:"secureSettings,omitempty"`
}

// LogstashHealth expresses the status of the Logstash instances.
type LogstashHealth string

const (
	// LogstashRed means no instance is currently available.
	LogstashRed LogstashHealth = "red"
	// LogstashGreen means at least one instance is available.
	LogstashGreen LogstashHealth = "green"
)

// LogstashStatus defines the observed state of Logstash
type LogstashStatus struct {
	commonv1alpha1.ReconcilerStatus `json:",inline"`
	Health                          LogstashHealth                   `json:"health,omitempty"`
	AssociationStatus               commonv1alpha1.AssociationStatus `json:"associationStatus,omitempty"`
}

// IsDegraded returns true if the current status is worse than the previous.
func (ls LogstashStatus) IsDegraded(prev LogstashStatus) bool {
	return prev.Health == LogstashGreen && ls.Health != LogstashGreen
}

// IsMarkedForDeletion returns true if the Logstash is going to be deleted
func (l Logstash) IsMarkedForDeletion() bool {
	return !l.DeletionTimestamp.IsZero()
}

func (l *Logstash) ElasticsearchRef() commonv1alpha1.ObjectSelector {
	return l.Spec.ElasticsearchRef
}

func (l *Logstash) SecureSettings() []commonv1alpha1.SecretSource {
	return l.Spec.SecureSettings
}

func (l *Logstash) AssociationConf() *commonv1alpha1.AssociationConf {
	return l.assocConf
}

func (l *Logstash) SetAssociationConf(assocConf *commonv1alpha1.AssociationConf) {
	l.assocConf = assocConf
}

// +kubebuilder:object:root=true

// Logstash is the Schema for the logstashes API
// +kubebuilder:resource:categories=elastic,shortName=kb
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="health",type="string",JSONPath=".status.health"
// +kubebuilder:printcolumn:name="nodes",type="integer",JSONPath=".status.availableNodes",description="Available nodes"
// +kubebuilder:printcolumn:name="version",type="string",JSONPath=".spec.version",description="Logstash version"
// +kubebuilder:printcolumn:name="age",type="date",JSONPath=".metadata.creationTimestamp"
type Logstash struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec      LogstashSpec                    `json:"spec,omitempty"`
	Status    LogstashStatus                  `json:"status,omitempty"`
	assocConf *commonv1alpha1.AssociationConf `json:"-"` //nolint:govet
}

// +kubebuilder:object:root=true

// LogstashList contains a list of Logstash
type LogstashList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Logstash `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Logstash{}, &LogstashList{})
}
