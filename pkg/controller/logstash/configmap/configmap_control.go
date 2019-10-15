// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package configmap

import (
	"reflect"

	"github.com/cloudptio/logstash-operator/pkg/apis/logstash/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/reconciler"
	"github.com/cloudptio/logstash-operator/pkg/utils/k8s"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ReconcileConfigMap checks for an existing config map and updates it or creates one if it does not exist.
func ReconcileConfigMap(
	c k8s.Client,
	scheme *runtime.Scheme,
	es v1beta1.Logstash,
	expected corev1.ConfigMap,
) error {
	reconciled := &corev1.ConfigMap{}
	return reconciler.ReconcileResource(
		reconciler.Params{
			Client:     c,
			Scheme:     scheme,
			Owner:      &es,
			Expected:   &expected,
			Reconciled: reconciled,
			NeedsUpdate: func() bool {
				return !reflect.DeepEqual(expected.Data, reconciled.Data)
			},
			UpdateReconciled: func() {
				reconciled.Data = expected.Data
			},
		},
	)
}
