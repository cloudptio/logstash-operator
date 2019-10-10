// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"reflect"

	"github.com/elastic/cloud-on-k8s/pkg/about"
	"github.com/elastic/cloud-on-k8s/pkg/apis/logstash/v1beta1"
	"github.com/elastic/cloud-on-k8s/pkg/controller/common/reconciler"
	"github.com/elastic/cloud-on-k8s/pkg/controller/logstash/label"
	"github.com/elastic/cloud-on-k8s/pkg/utils/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

// ReconcileConfigSecret reconciles the expected Logstash config secret for the given Logstash resource.
// This managed secret is mounted into each pod of the Logstash deployment.
func ReconcileConfigSecret(
	client k8s.Client,
	ls v1beta1.Logstash,
	lsSettings CanonicalConfig,
	operatorInfo about.OperatorInfo,
) error {
	settingsYamlBytes, err := lsSettings.Render()
	if err != nil {
		return err
	}
	telemetryYamlBytes, err := getTelemetryYamlBytes(operatorInfo)
	if err != nil {
		return err
	}
	expected := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ls.Namespace,
			Name:      SecretName(ls),
			Labels: map[string]string{
				label.LogstashNameLabelName: ls.Name,
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
		Owner:      &ls,
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
