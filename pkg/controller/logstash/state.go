// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logstash

import (
	"github.com/elastic/cloud-on-k8s/pkg/apis/logstash/v1beta1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// State holds the accumulated state during the reconcile loop including the response and a pointer to a Logstash
// resource for status updates.
type State struct {
	Logstash  *v1beta1.Logstash
	Request reconcile.Request

	originalLogstash *v1beta1.Logstash
}

// NewState creates a new reconcile state based on the given request and Logstash resource with the resource
// state reset to empty.
func NewState(request reconcile.Request, ls *v1beta1.Logstash) State {
	return State{Request: request, Logstash: ls, originalLogstash: ls.DeepCopy()}
}

// UpdateLogstashState updates the Logstash status based on the given deployment.
func (s State) UpdateLogstashState(deployment v1.Deployment) {
	s.Logstash.Status.AvailableNodes = int(deployment.Status.AvailableReplicas) // TODO lossy type conversion
	s.Logstash.Status.Health = v1beta1.LogstashRed
	for _, c := range deployment.Status.Conditions {
		if c.Type == v1.DeploymentAvailable && c.Status == corev1.ConditionTrue {
			s.Logstash.Status.Health = v1beta1.LogstashGreen
		}
	}
}
