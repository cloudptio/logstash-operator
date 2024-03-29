// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"reflect"

	"github.com/cloudptio/logstash-operator/pkg/about"
	"github.com/cloudptio/logstash-operator/pkg/apis/kibana/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/reconciler"
	"github.com/cloudptio/logstash-operator/pkg/controller/kibana/label"
	"github.com/cloudptio/logstash-operator/pkg/utils/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

// ReconcileConfigSecret reconciles the expected Kibana config secret for the given Kibana resource.
// This managed secret is mounted into each pod of the Kibana deployment.
func ReconcileConfigSecret(
	client k8s.Client,
	kb v1beta1.Kibana,
	kbSettings CanonicalConfig,
	operatorInfo about.OperatorInfo,
) error {
	settingsYamlBytes, err := kbSettings.Render()
	if err != nil {
		return err
	}
	telemetryYamlBytes, err := getTelemetryYamlBytes(operatorInfo)
	if err != nil {
		return err
	}
	expected := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: kb.Namespace,
			Name:      SecretName(kb),
			Labels: map[string]string{
				label.KibanaNameLabelName: kb.Name,
			},
		},
		Data: map[string][]byte{
			SettingsFilename:  settingsYamlBytes,
			telemetryFilename: telemetryYamlBytes,
		},
	}
	reconciled := corev1.Secret{}
	if err := reconciler.ReconcileResource(reconciler.Params{
		Client:     client,
		Scheme:     scheme.Scheme,
		Owner:      &kb,
		Expected:   &expected,
		Reconciled: &reconciled,
		NeedsUpdate: func() bool {
			return !reflect.DeepEqual(reconciled.Data, expected.Data)
		},
		UpdateReconciled: func() {
			reconciled.Data = expected.Data
		},
	}); err != nil {
		return err
	}
	return nil
}
